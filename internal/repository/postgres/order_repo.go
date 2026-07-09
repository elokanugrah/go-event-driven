package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/elokanugrah/go-event-driven/internal/domain"
	"github.com/elokanugrah/go-event-driven/internal/usecase"
)

// Ensure PostgresOrderRepository implements the usecase.OrderRepository interface.
var _ usecase.OrderRepository = (*PostgresOrderRepository)(nil)

// querier is an interface that is satisfied by both *sql.DB and *sql.Tx.
// This allows repository methods to work with or without a transaction.
type querier interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

type PostgresOrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *PostgresOrderRepository {
	return &PostgresOrderRepository{db: db}
}

// getQuerier extracts a transaction from the context if it exists,
// otherwise it returns the base database connection.
func (r *PostgresOrderRepository) getQuerier(ctx context.Context) querier {
	tx, ok := ctx.Value(txKey{}).(*sql.Tx)
	if ok {
		return tx
	}

	return r.db
}

// Save inserts a new order and its items into the database.
func (r *PostgresOrderRepository) Save(ctx context.Context, order *domain.Order) error {
	// Get the correct querier (either the transaction or the base DB connection).
	q := r.getQuerier(ctx)

	// Insert the main order record into the 'orders' table.
	// Use RETURNING to get the generated order ID back immediately.
	orderQuery := `INSERT INTO orders (user_id, total_amount, status, created_at, updated_at) 
                   VALUES ($1, $2, $3, $4, $5) 
                   RETURNING id, created_at, updated_at`

	now := time.Now()
	err := q.QueryRowContext(ctx, orderQuery,
		order.UserID,
		order.TotalAmount,
		order.Status,
		now,
		now,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error saving order: %w", err)
	}

	// Insert all order items into the 'order_items' table.
	itemQuery := `INSERT INTO order_items (order_id, product_id, quantity, price_at_order) VALUES `

	vals := []interface{}{}
	var placeholders []string

	for i, item := range order.OrderItems {
		p_num := i * 4
		placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d, $%d)", p_num+1, p_num+2, p_num+3, p_num+4))

		vals = append(vals, order.ID, item.Product.ID, item.Quantity, item.PriceAtOrder)
	}

	itemQuery += strings.Join(placeholders, ", ")
	itemQuery += " RETURNING id"

	rows, err := q.QueryContext(ctx, itemQuery, vals...)
	if err != nil {
		return fmt.Errorf("error saving order items: %w", err)
	}
	defer rows.Close()

	var newItemIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("error scanning returned order item id: %w", err)
		}
		newItemIDs = append(newItemIDs, id)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error after scanning returned ids: %w", err)
	}

	// Ensure we got the same number of IDs back as the items we inserted.
	if len(newItemIDs) != len(order.OrderItems) {
		return errors.New("mismatch in number of saved order items")
	}

	// Assign the new IDs and the OrderID back to the domain object.
	for i := range order.OrderItems {
		order.OrderItems[i].ID = newItemIDs[i]
		order.OrderItems[i].OrderID = order.ID
	}

	return nil
}
