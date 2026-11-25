package messaging

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	TaskQueue     = "tenant.tasks"
	ProgressQueue = "tenant.progress"
	TaskDLQ       = "tenant.tasks.dlq"
	ProgressDLQ   = "tenant.progress.dlq"
)

func SetupQueues(conn *amqp.Connection) error {
	ch, err := conn.Channel()

	if err != nil {
		return fmt.Errorf("create channel: %w", err)
	}

	defer ch.Close()

	// Declare dead letter queues first.
	for _, name := range []string{
		TaskDLQ,
		ProgressDLQ,
	} {
		if err := declareQueue(ch, name, nil); err != nil {
			return err
		}
	}

	// Declare task queue.
	err = declareQueue(
		ch,
		TaskQueue,
		amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": TaskDLQ,
		},
	)

	if err != nil {
		return err
	}

	// Declare worker progress queue.
	return declareQueue(
		ch,
		ProgressQueue,
		amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": ProgressDLQ,
		},
	)
}

func declareQueue(
	ch *amqp.Channel,
	name string,
	args amqp.Table,
) error {

	_, err := ch.QueueDeclare(
		name,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		args,
	)

	if err != nil {
		return fmt.Errorf(
			"declare queue %s: %w",
			name,
			err,
		)
	}

	return nil
}
