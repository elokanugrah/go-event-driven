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

// FindByID retrieves a single order and its items from PostgreSQL.
func (r *PostgresOrderRepository) FindByID(ctx context.Context, id int64) (*domain.Order, error) {
	q := r.getQuerier(ctx)

	query := `SELECT id, user_id, total_amount, status, created_at, updated_at FROM orders WHERE id = $1`
	var order domain.Order
	err := q.QueryRowContext(ctx, query, id).Scan(
		&order.ID, &order.UserID, &order.TotalAmount, &order.Status, &order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error finding order: %w", err)
	}

	itemsQuery := `SELECT oi.id, oi.order_id, oi.quantity, oi.price_at_order, p.id, p.name, p.price, p.quantity, p.created_at, p.updated_at
				   FROM order_items oi
				   JOIN products p ON oi.product_id = p.id
				   WHERE oi.order_id = $1`
	rows, err := q.QueryContext(ctx, itemsQuery, id)
	if err != nil {
		return nil, fmt.Errorf("error finding order items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(
			&item.ID, &item.OrderID, &item.Quantity, &item.PriceAtOrder,
			&item.Product.ID, &item.Product.Name, &item.Product.Price, &item.Product.Quantity, &item.Product.CreatedAt, &item.Product.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("error scanning order item row: %w", err)
		}
		order.OrderItems = append(order.OrderItems, item)
	}

	return &order, nil
}

// FindAll retrieves paginated orders and their items.
func (r *PostgresOrderRepository) FindAll(ctx context.Context, limit, offset int) ([]domain.Order, error) {
	q := r.getQuerier(ctx)

	query := `SELECT id, user_id, total_amount, status, created_at, updated_at 
			  FROM orders 
			  ORDER BY id DESC 
			  LIMIT $1 OFFSET $2`

	rows, err := q.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error listing orders: %w", err)
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var order domain.Order
		err := rows.Scan(
			&order.ID, &order.UserID, &order.TotalAmount, &order.Status, &order.CreatedAt, &order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning order row: %w", err)
		}
		orders = append(orders, order)
	}

	if len(orders) == 0 {
		return orders, nil
	}

	// Fetch all items for the retrieved orders in one query.
	var orderIDs []string
	idMap := make(map[int64]int)
	for i, o := range orders {
		orderIDs = append(orderIDs, fmt.Sprintf("%d", o.ID))
		idMap[o.ID] = i
	}

	itemsQuery := fmt.Sprintf(`
		SELECT oi.id, oi.order_id, oi.quantity, oi.price_at_order, p.id, p.name, p.price, p.quantity, p.created_at, p.updated_at
		FROM order_items oi
		JOIN products p ON oi.product_id = p.id
		WHERE oi.order_id IN (%s)
	`, strings.Join(orderIDs, ","))

	itemRows, err := q.QueryContext(ctx, itemsQuery)
	if err != nil {
		return nil, fmt.Errorf("error querying order items list: %w", err)
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var item domain.OrderItem
		if err := itemRows.Scan(
			&item.ID, &item.OrderID, &item.Quantity, &item.PriceAtOrder,
			&item.Product.ID, &item.Product.Name, &item.Product.Price, &item.Product.Quantity, &item.Product.CreatedAt, &item.Product.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("error scanning order item row: %w", err)
		}
		idx := idMap[item.OrderID]
		orders[idx].OrderItems = append(orders[idx].OrderItems, item)
	}

	return orders, nil
}

// Update updates an order's status and updated timestamp.
func (r *PostgresOrderRepository) Update(ctx context.Context, order *domain.Order) error {
	q := r.getQuerier(ctx)

	query := `UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3`
	result, err := q.ExecContext(ctx, query, order.Status, time.Now(), order.ID)
	if err != nil {
		return fmt.Errorf("error updating order: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return errors.New("order not found for update")
	}

	return nil
}
