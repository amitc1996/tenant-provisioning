package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/amitc1996/tenant-provisioning/internal/application"
	"github.com/amitc1996/tenant-provisioning/internal/domain"
	"github.com/amitc1996/tenant-provisioning/internal/messaging"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type ProgressConsumer struct {
	url     string
	service *application.Service
}

func NewProgressConsumer(
	url string,
	service *application.Service,
) *ProgressConsumer {

	return &ProgressConsumer{
		url:     url,
		service: service,
	}
}

func (p *ProgressConsumer) Run(
	ctx context.Context,
) {

	delay := time.Second

	for {
		if ctx.Err() != nil {
			return
		}

		started := time.Now()

		err := p.consume(ctx)

		if ctx.Err() != nil {
			return
		}

		if time.Since(started) >= 30*time.Second {
			delay = time.Second
		}

		log.Printf(
			"progress consumer interrupted: %v; retry in %s",
			err,
			delay,
		)

		timer := time.NewTimer(delay)

		select {
		case <-ctx.Done():
			timer.Stop()
			return

		case <-timer.C:
		}

		delay *= 2

		if delay > 30*time.Second {
			delay = 30 * time.Second
		}
	}
}

func (p *ProgressConsumer) consume(
	ctx context.Context,
) error {

	conn, err := messaging.Connect(p.url)

	if err != nil {
		return err
	}

	defer conn.Close()

	if err := messaging.SetupQueues(conn); err != nil {
		return err
	}

	ch, err := conn.Channel()

	if err != nil {
		return err
	}

	defer ch.Close()

	if err := ch.Qos(1, 0, false); err != nil {
		return err
	}

	deliveries, err := ch.Consume(
		messaging.ProgressQueue,
		"",
		false, // Manual ACK
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return err
	}

	log.Println(
		"Progress consumer listening on",
		messaging.ProgressQueue,
	)

	for {
		select {

		case <-ctx.Done():
			return ctx.Err()

		case delivery, ok := <-deliveries:

			if !ok {
				return errors.New(
					"progress delivery channel closed",
				)
			}

			if err := p.handleDelivery(
				ctx,
				delivery,
			); err != nil {
				return err
			}
		}
	}
}

func (p *ProgressConsumer) handleDelivery(
	ctx context.Context,
	delivery amqp.Delivery,
) error {

	var event messaging.ProgressEvent

	if err := json.Unmarshal(
		delivery.Body,
		&event,
	); err != nil {

		log.Println("Rejecting invalid progress JSON")

		return delivery.Reject(false)
	}

	if !validProgressEvent(event) {

		log.Println("Rejecting invalid progress event")

		return delivery.Reject(false)
	}

	err := p.service.ApplyTaskProgress(
		ctx,
		application.ProgressInput{
			EventID: event.ID,
			TaskID:  event.TaskID,
			Status: domain.TaskStatus(
				event.Status,
			),
		},
	)

	if err != nil {

		// Permanent domain errors go to the DLQ.
		if isPermanentError(err) {

			log.Printf(
				"rejecting progress event: id=%s error=%v",
				event.ID,
				err,
			)

			return delivery.Reject(false)
		}

		// Transient failure: do not acknowledge.
		// Closing the connection requeues the delivery.
		return err
	}

	// Database transaction has committed successfully.
	if err := delivery.Ack(false); err != nil {
		return err
	}

	log.Printf(
		"progress event handled: id=%s status=%s",
		event.ID,
		event.Status,
	)

	return nil
}

func validProgressEvent(
	event messaging.ProgressEvent,
) bool {

	if strings.TrimSpace(event.ID) == "" ||
		len(event.ID) > 200 ||
		strings.ContainsRune(event.ID, 0) {
		return false
	}

	taskID, err := uuid.Parse(event.TaskID)

	if err != nil ||
		taskID.String() != event.TaskID {
		return false
	}

	switch event.Status {
	case "in_progress", "done", "failed":
		return true

	default:
		return false
	}
}

func isPermanentError(err error) bool {
	return errors.Is(err, domain.ErrTaskNotFound) ||
		errors.Is(err, domain.ErrTenantNotFound) ||
		errors.Is(err, domain.ErrInvalidTaskTransition) ||
		errors.Is(err, domain.ErrInvalidTenantTransition) ||
		errors.Is(err, domain.ErrInvalidTaskType)
}
