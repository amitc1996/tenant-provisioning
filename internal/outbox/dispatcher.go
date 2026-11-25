package outbox

import (
	"context"
	"log"
	"time"

	"github.com/amitc1996/tenant-provisioning/internal/messaging"
	"github.com/amitc1996/tenant-provisioning/internal/repository"
)

type Dispatcher struct {
	repo      *repository.Repository
	publisher *messaging.Publisher
}

func New(
	repo *repository.Repository,
	publisher *messaging.Publisher,
) *Dispatcher {

	return &Dispatcher{
		repo:      repo,
		publisher: publisher,
	}
}

func (d *Dispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(
		500 * time.Millisecond,
	)

	defer ticker.Stop()
	defer d.publisher.Close()

	log.Println("Outbox dispatcher started")

	for {
		select {

		case <-ctx.Done():
			return

		case <-ticker.C:
			if err := d.dispatch(ctx); err != nil {
				log.Printf(
					"outbox dispatch failed: %v",
					err,
				)
			}
		}
	}
}

func (d *Dispatcher) dispatch(
	ctx context.Context,
) error {

	return d.repo.WithinTx(
		ctx,
		func(q *repository.Queries) error {

			event, err := q.GetNextOutboxEvent(ctx)

			if err != nil {
				return err
			}

			if event == nil {
				return nil
			}

			err = d.publisher.Publish(
				ctx,
				messaging.TaskQueue,
				event.TaskID,
				event.Payload,
			)

			if err != nil {

				delay := retryDelay(event.Attempts)

				log.Printf(
					"publish failed: task=%s retry=%ds",
					event.TaskID,
					delay,
				)

				return q.RetryOutboxEvent(
					ctx,
					event.TaskID,
					delay,
				)
			}

			return q.MarkOutboxPublished(
				ctx,
				event.TaskID,
			)
		},
	)
}

func retryDelay(attempts int) int {
	delay := 1

	for i := 0; i < attempts && delay < 60; i++ {
		delay *= 2
	}

	if delay > 60 {
		return 60
	}

	return delay
}
