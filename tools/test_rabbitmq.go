package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"core_project/shared/sdk/queue"

	"github.com/google/uuid"
)

func main() {
	// Connect to RabbitMQ
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://wch_admin:rabbitmq_pass@127.0.0.1:10672/"
	}

	fmt.Printf("Connecting to RabbitMQ: %s\n", rabbitURL)
	client, err := queue.NewClient(rabbitURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	fmt.Println("✓ Connected to RabbitMQ")

	// Declare test queue
	testQueue := "test.queue"
	if err := client.DeclareQueue(testQueue); err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}
	fmt.Printf("✓ Queue '%s' declared\n", testQueue)

	// Publish test message
	job := queue.Job{
		JobID:     uuid.New().String(),
		TenantID:  "test-tenant-id",
		Type:      "test.job",
		Data:      map[string]interface{}{"message": "Hello from RabbitMQ test!"},
		CreatedAt: time.Now(),
	}

	ctx := context.Background()
	if err := client.Publish(ctx, testQueue, job); err != nil {
		log.Fatalf("Failed to publish: %v", err)
	}
	fmt.Printf("✓ Published job: %s\n", job.JobID)

	// Consume test message (with timeout)
	consumed := make(chan bool)
	go func() {
		client.Consume(testQueue, func(j queue.Job) error {
			fmt.Printf("✓ Consumed job: %s (type=%s, data=%v)\n", j.JobID, j.Type, j.Data)
			consumed <- true
			return nil
		})
	}()

	select {
	case <-consumed:
		fmt.Println("\n✅ RabbitMQ integration test passed!")
	case <-time.After(5 * time.Second):
		log.Fatal("❌ Timeout waiting for message")
	}
}
