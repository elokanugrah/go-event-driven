package usecase

import (
	"context"
	"log"
	"time"

	"github.com/elokanugrah/go-event-driven/internal/domain"
)

type OutboxPublisher struct {
	outboxRepo OutboxRepository
	txManager  TransactionManager
	broker     MessageBroker
	interval   time.Duration
}

func NewOutboxPublisher(
	or OutboxRepository,
	tm TransactionManager,
	mb MessageBroker,
	interval time.Duration,
) *OutboxPublisher {
	return &OutboxPublisher{
		outboxRepo: or,
		txManager:  tm,
		broker:     mb,
		interval:   interval,
	}
}

// Start runs the outbox publisher loop. It blocks until the context is canceled.
func (p *OutboxPublisher) Start(ctx context.Context) {
	log.Println("Starting Outbox Publisher background worker...")
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping Outbox Publisher background worker...")
			return
		case <-ticker.C:
			p.ProcessEvents(ctx)
		}
	}
}

// ProcessEvents checks for pending events and publishes them to the message broker.
// It wraps the operations in a database transaction to ensure SKIP LOCKED locking is active.
func (p *OutboxPublisher) ProcessEvents(ctx context.Context) {
	err := p.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// Fetch a batch of pending events.
		events, err := p.outboxRepo.FindPending(txCtx, 10)
		if err != nil {
			return err
		}

		if len(events) == 0 {
			return nil
		}

		for _, event := range events {
			// Publish the event to RabbitMQ using the transaction context.
			err = p.broker.Publish(txCtx, event.EventType, event.Payload)
			if err != nil {
				// Return error to rollback transaction and retry this batch later.
				log.Printf("[OUTBOX] ERROR: Failed to publish event %d (%s): %v. Rolling back batch.", event.ID, event.EventType, err)
				return err
			}

			// Mark the event as processed.
			err = p.outboxRepo.UpdateStatus(txCtx, event.ID, domain.OutboxStatusProcessed)
			if err != nil {
				return err
			}
			log.Printf("[OUTBOX] Successfully processed event %d (%s)", event.ID, event.EventType)
		}

		return nil
	})

	if err != nil {
		log.Printf("[OUTBOX] Batch process encountered an error: %v", err)
	}
}
