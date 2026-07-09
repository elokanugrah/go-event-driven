package usecase

import (
	"context"

	"github.com/elokanugrah/go-event-driven/internal/domain"
)

//go:generate mockery --name ProductRepository --output ./mocks --case=snake
type ProductRepository interface {
	// Create
	Save(ctx context.Context, product *domain.Product) error

	// Read
	FindByID(ctx context.Context, id int64) (*domain.Product, error)
	FindManyByIDs(ctx context.Context, ids []int64) ([]domain.Product, error)
	FindAll(ctx context.Context, limit, offset int) ([]domain.Product, error)

	// Update
	Update(ctx context.Context, product *domain.Product) error

	// Delete
	Delete(ctx context.Context, id int64) error
}

//go:generate mockery --name OrderRepository --output ./mocks --case=snake
type OrderRepository interface {
	// Create
	Save(ctx context.Context, order *domain.Order) error
}

// TransactionManager defines the contract for database transaction management.
// This allows use cases to run operations within a single transaction
// without being coupled to a specific database implementation.
//
//go:generate mockery --name TransactionManager --output ./mocks --case=snake
type TransactionManager interface {
	WithTransaction(ctx context.Context, fn func(txCtx context.Context) error) error
}

//go:generate mockery --name MessageBroker --output ./mocks --case=snake
type MessageBroker interface {
	Publish(ctx context.Context, queueName string, message []byte) error
}
