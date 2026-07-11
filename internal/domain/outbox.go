package domain

import "time"

type OutboxStatus string

const (
	OutboxStatusPending   OutboxStatus = "pending"
	OutboxStatusProcessed OutboxStatus = "processed"
	OutboxStatusFailed    OutboxStatus = "failed"
)

type OutboxEvent struct {
	ID          int64
	EventType   string
	Payload     []byte
	Status      OutboxStatus
	CreatedAt   time.Time
	ProcessedAt *time.Time
}
