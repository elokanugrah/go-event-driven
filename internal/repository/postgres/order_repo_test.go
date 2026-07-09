package postgres_test

import (
	"context"
	"database/sql"
	"log"
	"testing"

	"github.com/elokanugrah/go-event-driven/internal/config"
	"github.com/elokanugrah/go-event-driven/internal/database"
	"github.com/elokanugrah/go-event-driven/internal/domain"
	"github.com/elokanugrah/go-event-driven/internal/repository/postgres"
	"github.com/stretchr/testify/suite"
)

type OrderRepositorySuite struct {
	suite.Suite
	db          *sql.DB
	orderRepo   *postgres.PostgresOrderRepository
	productRepo *postgres.PostgresProductRepository
}

// txKey is a private key type to store the transaction object in the context.
type txKey struct{}

// SetupSuite runs once before all tests in this suite.
// It's used for setting up the database connection.
func (s *OrderRepositorySuite) SetupSuite() {
	cfg := config.Load()
	s.db = database.NewConnection(cfg)
	s.orderRepo = postgres.NewOrderRepository(s.db)
	s.productRepo = postgres.NewProductRepository(s.db)
}

// TearDownSuite runs once after all tests in this suite are finished.
func (s *OrderRepositorySuite) TearDownSuite() {
	if err := s.db.Close(); err != nil {
		log.Fatalf("Failed to close test database connection: %v", err)
	}
}

// TearDownTest runs after each test function.
// It cleans all relevant tables to ensure test isolation.
func (s *OrderRepositorySuite) TearDownTest() {
	_, err := s.db.Exec("TRUNCATE TABLE order_items, orders, products RESTART IDENTITY CASCADE")
	s.Suite.NoError(err)
}

// This function is the entry point for running the test suite.
func TestOrderRepository(t *testing.T) {
	suite.Run(t, new(OrderRepositorySuite))
}

// TestSave tests the full process of saving an order and its items.
func (s *OrderRepositorySuite) TestSave() {
	assert := s.Suite.Assert()
	ctx := context.Background()

	product1 := &domain.Product{Name: "Laptop", Price: 15000000, Quantity: 10}
	product2 := &domain.Product{Name: "Mouse", Price: 500000, Quantity: 20}
	err := s.productRepo.Save(ctx, product1)
	assert.NoError(err)
	err = s.productRepo.Save(ctx, product2)
	assert.NoError(err)

	// Note: the IDs for items are still 0.
	orderToSave := &domain.Order{
		UserID: 123,
		Status: domain.StatusPending,
		OrderItems: []domain.OrderItem{
			{Product: *product1, Quantity: 1, PriceAtOrder: product1.Price},
			{Product: *product2, Quantity: 2, PriceAtOrder: product2.Price},
		},
	}

	orderToSave.CalculateTotalAmount()
	expectedTotal := (1 * 15000000.0) + (2 * 500000.0)
	assert.Equal(expectedTotal, orderToSave.TotalAmount)

	tx, err := s.db.Begin()
	assert.NoError(err)
	defer tx.Rollback()

	txCtx := context.WithValue(ctx, txKey{}, tx)

	err = s.orderRepo.Save(txCtx, orderToSave)

	// Assert
	assert.NoError(err)

	// Assert that the domain object in memory is now updated with new IDs.
	assert.NotZero(orderToSave.ID)
	assert.NotZero(orderToSave.OrderItems[0].ID)
	assert.Equal(orderToSave.ID, orderToSave.OrderItems[0].OrderID)
	assert.NotZero(orderToSave.OrderItems[1].ID)
	assert.Equal(orderToSave.ID, orderToSave.OrderItems[1].OrderID)

	// Final verification by query database directly

	// Verify the 'orders' table
	var dbUserID int64
	var dbTotal float64
	err = tx.QueryRowContext(ctx, "SELECT user_id, total_amount FROM orders WHERE id = $1", orderToSave.ID).Scan(&dbUserID, &dbTotal)
	assert.NoError(err)
	assert.Equal(int64(123), dbUserID)
	assert.Equal(expectedTotal, dbTotal)

	// Verify the 'order_items' table
	rows, err := tx.QueryContext(ctx, "SELECT product_id, quantity FROM order_items WHERE order_id = $1 ORDER BY product_id", orderToSave.ID)
	assert.NoError(err)
	defer rows.Close()

	var dbItems []domain.OrderItem
	for rows.Next() {
		var item domain.OrderItem
		err := rows.Scan(&item.Product.ID, &item.Quantity)
		assert.NoError(err)
		dbItems = append(dbItems, item)
	}

	assert.Len(dbItems, 2) // Expectation two items to be saved
	assert.Equal(product1.ID, dbItems[0].Product.ID)
	assert.Equal(1, dbItems[0].Quantity)
	assert.Equal(product2.ID, dbItems[1].Product.ID)
	assert.Equal(2, dbItems[1].Quantity)

	tx.Commit()
}
