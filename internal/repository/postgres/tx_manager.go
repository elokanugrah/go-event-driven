package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/elokanugrah/go-event-driven/internal/usecase"
)

// txKey is the key used to store the transaction object in the context.
type txKey struct{}

type PostgresTransactionManager struct {
	db *sql.DB
}

func NewTransactionManager(db *sql.DB) usecase.TransactionManager {
	return &PostgresTransactionManager{db: db}
}

// WithTransaction executes a function within a database transaction.
// It begins a transaction, calls the provided function with a new context
// containing the transaction, and then commits or rolls back based on the error.
func (tm *PostgresTransactionManager) WithTransaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	tx, err := tm.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create a new context with the transaction object
	txCtx := context.WithValue(ctx, txKey{}, tx)

	// Use a deferred function to recover from panics and rollback the transaction.
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p) // re-panic after rolling back
		}
	}()

	// Execute the provided function with the transaction context
	err = fn(txCtx)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction failed: %v, unable to rollback: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}
