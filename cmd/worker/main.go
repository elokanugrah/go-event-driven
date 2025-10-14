package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/elokanugrah/go-order-system/internal/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	log.Println("Starting Worker Service...")

	cfg := config.Load()

	// Connect to RabbitMQ
	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	// Declare the queue to make sure it exists
	q, err := ch.QueueDeclare(
		"orders.created",
		true, false, false, false, nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	// Start consuming messages from the queue
	msgs, err := ch.Consume(
		q.Name,
		"order-worker", // consumer name
		false,          // auto-ack
		false,          // exclusive
		false,          // no-local
		false,          // no-wait
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	// Goroutine to process messages
	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
			if err := processMessage(d.Body); err != nil {
				log.Printf("ERROR: Failed to process message: %v. Message will be requeued.", err)
				// Nack (Negative Acknowledge) - let RabbitMQ know message failed to process
				// First param: multiple (false, only for this message)
				// Second param: requeue (true, return message to queue)
				if nackErr := d.Nack(false, true); nackErr != nil {
					log.Printf("ERROR: Failed to Nack message: %v", nackErr)
				}
			} else {
				// Ack (Acknowledge) - let RabbitMQ know message process successfully
				// Parameter: multiple (false, only for this message)
				if ackErr := d.Ack(false); ackErr != nil {
					log.Printf("ERROR: Failed to Ack message: %v", ackErr)
				}
			}
		}
	}()

	log.Printf("Worker is waiting for messages. To exit press CTRL+C")

	// Handles graceful shutdown on receiving SIGINT or SIGTERM signals.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down worker...")

	log.Println("Worker exited gracefully.")
}

// A helper function to process the message payload.
func processMessage(body []byte) error {
	time.Sleep(2 * time.Second) // Simulate a 2-second task

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("[WORKER] ERROR: Failed to unmarshal message: %v", err)
		return err
	}

	if orderID, ok := payload["order_id"]; ok {
		log.Printf("[WORKER] Finished processing confirmation for Order ID: %.0f", orderID)
	} else {
		log.Println("[WORKER] WARNING: order_id not found in message payload.")
	}

	// Example failed simulation
	// if rand.Intn(10) < 3 { // 30% chance of failure
	//     return errors.New("simulated processing failure")
	// }

	return nil
}
