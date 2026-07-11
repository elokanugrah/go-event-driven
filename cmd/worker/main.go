package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/elokanugrah/go-event-driven/internal/config"
	"github.com/elokanugrah/go-event-driven/internal/database"
	"github.com/elokanugrah/go-event-driven/internal/domain"
	"github.com/elokanugrah/go-event-driven/internal/repository/postgres"
	"github.com/elokanugrah/go-event-driven/internal/usecase"
	amqp "github.com/rabbitmq/amqp091-go"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("Starting Worker Service...")

	cfg := config.Load()

	// Connect to database
	db := database.NewConnection(cfg)
	defer db.Close()

	orderRepo := postgres.NewOrderRepository(db)

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
		ctx := context.Background()
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
			if err := processMessage(ctx, d.Body, orderRepo); err != nil {
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

// A helper function to process the message payload and update the database order status.
func processMessage(ctx context.Context, body []byte, orderRepo usecase.OrderRepository) error {
	time.Sleep(2 * time.Second) // Simulate a 2-second task

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("[WORKER] ERROR: Failed to unmarshal message: %v", err)
		return err
	}

	orderIDVal, ok := payload["order_id"]
	if !ok {
		log.Println("[WORKER] WARNING: order_id not found in message payload.")
		return errors.New("order_id not found in message payload")
	}

	orderID := int64(orderIDVal.(float64))
	log.Printf("[WORKER] Processing Order ID: %d", orderID)

	// Fetch order from database to ensure it exists and get latest details
	order, err := orderRepo.FindByID(ctx, orderID)
	if err != nil {
		log.Printf("[WORKER] ERROR: Failed to fetch order %d from database: %v", orderID, err)
		return err // Return error so it gets Nacked/requeued (system failure)
	}

	if order == nil {
		log.Printf("[WORKER] WARNING: Order %d not found in database. Discarding message.", orderID)
		return nil // Ack to discard since it's a permanent logical failure
	}

	// Example failed simulation (30% chance of business failure)
	x := rand.Intn(10)
	if x < 3 {
		log.Printf("[WORKER] Simulated business processing failure for Order ID: %d", orderID)
		
		// Update status to cancelled in the DB
		order.ChangeStatus(domain.StatusCancelled)
		if updateErr := orderRepo.Update(ctx, order); updateErr != nil {
			log.Printf("[WORKER] ERROR: Failed to update status to cancelled for Order ID: %d: %v", orderID, updateErr)
			return updateErr // Return error to requeue (DB update failed)
		}
		
		// Return nil so it gets Acked and removed from queue (handled business failure)
		return nil
	}

	// Successful processing
	order.ChangeStatus(domain.StatusCompleted)
	if updateErr := orderRepo.Update(ctx, order); updateErr != nil {
		log.Printf("[WORKER] ERROR: Failed to update status to completed for Order ID: %d: %v", orderID, updateErr)
		return updateErr // Return error to requeue
	}

	log.Printf("[WORKER] Finished processing confirmation for Order ID: %d (Status: Completed)", orderID)
	return nil
}
