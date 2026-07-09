package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/elokanugrah/go-event-driven/internal/domain"
	"github.com/elokanugrah/go-event-driven/internal/usecase"
	"github.com/lib/pq"
)

var _ usecase.ProductRepository = (*PostgresProductRepository)(nil)

type PostgresProductRepository struct {
	db *sql.DB
}

// Save inserts a new product into the database.
func (r *PostgresProductRepository) Save(ctx context.Context, product *domain.Product) error {
	query := `INSERT INTO products (name, price, quantity, created_at, updated_at) 
			   VALUES ($1, $2, $3, $4, $5) 
			   RETURNING id, created_at, updated_at`

	now := time.Now()
	err := r.db.QueryRowContext(ctx, query,
		product.Name,
		product.Price,
		product.Quantity,
		now,
		now,
	).Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt)

	if err != nil {
		return fmt.Errorf("error saving product: %w", err)
	}

	return nil
}

// Update modifies an existing product in the database.
func (r *PostgresProductRepository) Update(ctx context.Context, product *domain.Product) error {
	query := `UPDATE products 
			   SET name = $1, price = $2, quantity = $3, updated_at = $4 
			   WHERE id = $5`

	result, err := r.db.ExecContext(ctx, query,
		product.Name,
		product.Price,
		product.Quantity,
		time.Now(),
		product.ID,
	)
	if err != nil {
		return fmt.Errorf("error updating product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return errors.New("product not found for update")
	}

	return nil
}

// Delete removes a product from the database by its ID.
func (r *PostgresProductRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM products WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("error deleting product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return errors.New("product not found for delete")
	}

	return nil
}

// FindAll retrieves a paginated list of all products.
func (r *PostgresProductRepository) FindAll(ctx context.Context, limit int, offset int) ([]domain.Product, error) {
	query := `SELECT id, name, price, quantity, created_at, updated_at 
			   FROM products 
			   ORDER BY id ASC 
			   LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error querying products: %w", err)
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Quantity, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("error scanning product row: %w", err)
		}
		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	return products, nil
}

// FindByID retrieves a single product from the database by its ID.
func (r *PostgresProductRepository) FindByID(ctx context.Context, id int64) (*domain.Product, error) {
	query := `SELECT id, name, price, quantity, created_at, updated_at FROM products WHERE id = $1`
	var p domain.Product

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.Price, &p.Quantity, &p.CreatedAt, &p.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Return nil, nil to indicate not found, use case will handle it.
		}
		return nil, fmt.Errorf("error scanning product: %w", err)
	}

	return &p, nil
}

// FindManyByIDs retrieves multiple products based on a slice of IDs.
func (r *PostgresProductRepository) FindManyByIDs(ctx context.Context, ids []int64) ([]domain.Product, error) {
	query := `SELECT id, name, price, quantity, created_at, updated_at 
			   FROM products 
			   WHERE id = ANY($1)`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("error querying products by ids: %w", err)
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Quantity, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("error scanning product row: %w", err)
		}
		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	return products, nil
}

func NewProductRepository(db *sql.DB) *PostgresProductRepository {
	return &PostgresProductRepository{db: db}
}
