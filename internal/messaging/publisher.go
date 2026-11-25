package messaging

import (
	"context"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	url  string
	conn *amqp.Connection
}

func NewPublisher(url string) *Publisher {
	return &Publisher{
		url: url,
	}
}

func (p *Publisher) ensureConnection() error {
	if p.conn != nil && !p.conn.IsClosed() {
		return nil
	}

	conn, err := Connect(p.url)

	if err != nil {
		return err
	}

	if err := SetupQueues(conn); err != nil {
		_ = conn.Close()
		return err
	}

	p.conn = conn

	return nil
}

func (p *Publisher) Close() {
	if p.conn != nil {
		_ = p.conn.Close()
		p.conn = nil
	}
}

func (p *Publisher) Publish(
	ctx context.Context,
	queue string,
	eventID string,
	payload []byte,
) error {

	if err := p.ensureConnection(); err != nil {
		return err
	}

	err := p.publish(
		ctx,
		queue,
		eventID,
		payload,
	)

	if err != nil {
		p.Close()
	}

	return err
}

func (p *Publisher) publish(
	ctx context.Context,
	queue string,
	eventID string,
	payload []byte,
) error {

	ch, err := p.conn.Channel()

	if err != nil {
		return err
	}

	defer ch.Close()

	if err := ch.Confirm(false); err != nil {
		return err
	}

	returns := ch.NotifyReturn(
		make(chan amqp.Return, 1),
	)

	confirms := ch.NotifyPublish(
		make(chan amqp.Confirmation, 1),
	)

	ctx, cancel := context.WithTimeout(
		ctx,
		5*time.Second,
	)

	defer cancel()

	err = ch.PublishWithContext(
		ctx,
		"",    // Default exchange
		queue, // Routing key
		true,  // Mandatory
		false, // Immediate

		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    eventID,
			Timestamp:    time.Now().UTC(),
			Body:         payload,
		},
	)

	if err != nil {
		return err
	}

	select {

	case returned, ok := <-returns:
		if !ok {
			return errors.New("return channel closed")
		}

		return fmt.Errorf(
			"message returned: %s",
			returned.ReplyText,
		)

	case confirmation, ok := <-confirms:
		if !ok || !confirmation.Ack {
			return errors.New("message not confirmed")
		}
		select {
		case returned, ok := <-returns:
			if !ok {
				return errors.New("return channel closed")
			}

			return fmt.Errorf(
				"message returned: %s",
				returned.ReplyText,
			)

		default:
		}

		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}
