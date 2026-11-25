package messaging

import (
	"errors"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func Connect(url string) (*amqp.Connection, error) {
	if url == "" {
		return nil, errors.New("RABBITMQ_URL is required")
	}

	conn, err := amqp.Dial(url)

	if err != nil {
		return nil, fmt.Errorf(
			"connect to RabbitMQ: %w",
			err,
		)
	}

	return conn, nil
}
