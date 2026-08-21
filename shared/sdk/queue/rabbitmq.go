package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Client struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

type Job struct {
	JobID     string                 `json:"job_id"`
	TenantID  string                 `json:"tenant_id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	CreatedAt time.Time              `json:"created_at"`
}

func NewClient(url string) (*Client, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Set QoS to limit unacked messages per worker
	if err := ch.Qos(10, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	return &Client{conn: conn, channel: ch}, nil
}

func (c *Client) Close() error {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) DeclareQueue(name string) error {
	_, err := c.channel.QueueDeclare(
		name,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	return err
}

func (c *Client) Publish(ctx context.Context, queueName string, job Job) error {
	if err := c.DeclareQueue(queueName); err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	body, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	return c.channel.PublishWithContext(
		ctx,
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)
}

func (c *Client) Consume(queueName string, handler func(Job) error) error {
	if err := c.DeclareQueue(queueName); err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	msgs, err := c.channel.Consume(
		queueName,
		"",    // consumer tag
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	slog.Info("Worker started", "queue", queueName)

	for msg := range msgs {
		var job Job
		if err := json.Unmarshal(msg.Body, &job); err != nil {
			slog.Error("Failed to unmarshal job", "error", err)
			msg.Nack(false, false) // discard malformed message
			continue
		}

		slog.Info("Processing job", "job_id", job.JobID, "type", job.Type)

		if err := handler(job); err != nil {
			slog.Error("Job failed", "job_id", job.JobID, "error", err)
			msg.Nack(false, true) // requeue on failure
		} else {
			msg.Ack(false)
		}
	}

	return nil
}
