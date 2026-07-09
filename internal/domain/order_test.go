package domain_test

import (
	"testing"
	"time"

	"github.com/elokanugrah/go-event-driven/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestNewOrder(t *testing.T) {
	product1 := domain.Product{ID: 1, Price: 10000}
	product2 := domain.Product{ID: 2, Price: 5000}

	t.Run("should create a new order successfully with valid items", func(t *testing.T) {
		items := []domain.OrderItem{
			{Product: product1, Quantity: 2, PriceAtOrder: 10000}, // 20000
			{Product: product2, Quantity: 3, PriceAtOrder: 5000},  // 15000
		}
		userID := int64(123)
		expectedTotal := float64(35000)

		// Act
		order, err := domain.NewOrder(userID, items)

		assert.NoError(t, err)
		assert.NotNil(t, order)
		assert.Equal(t, userID, order.UserID)
		assert.Len(t, order.OrderItems, 2)
		assert.Equal(t, domain.StatusPending, order.Status) // Check default status
		assert.Equal(t, expectedTotal, order.TotalAmount)   // Check if total is calculated correctly
		assert.NotZero(t, order.CreatedAt)                  // Check if timestamp is set
		assert.NotZero(t, order.UpdatedAt)
	})

	t.Run("should return an error when created with no items", func(t *testing.T) {
		var emptyItems []domain.OrderItem
		userID := int64(123)

		// Act
		order, err := domain.NewOrder(userID, emptyItems)

		// Assert
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyOrder) // Check for the specific error
		assert.Nil(t, order)
	})
}

func TestOrder_CalculateTotalAmount(t *testing.T) {
	order := &domain.Order{
		OrderItems: []domain.OrderItem{
			{PriceAtOrder: 25.5, Quantity: 2}, // 51.0
			{PriceAtOrder: 10.0, Quantity: 5}, // 50.0
		},
	}
	expectedTotal := 101.0

	// Act
	order.CalculateTotalAmount()

	// Assert
	assert.Equal(t, expectedTotal, order.TotalAmount)
}

func TestOrder_AddItem(t *testing.T) {
	// Create an initial order
	order := &domain.Order{
		OrderItems: []domain.OrderItem{
			{Product: domain.Product{ID: 1}, PriceAtOrder: 100, Quantity: 1}, // Total = 100
		},
		TotalAmount: 100,
	}

	// Define the new item to add
	newItem := domain.OrderItem{
		Product:      domain.Product{ID: 2},
		PriceAtOrder: 50,
		Quantity:     2, // Value = 100
	}
	expectedTotal := 200.0

	// Act
	order.AddItem(newItem)

	// Assert
	assert.Len(t, order.OrderItems, 2)                // Check if item count is now 2
	assert.Equal(t, expectedTotal, order.TotalAmount) // Check if total amount was recalculated
}

func TestOrder_ChangeStatus(t *testing.T) {
	// Arrange
	order := &domain.Order{
		Status:    domain.StatusPending,
		UpdatedAt: time.Now(),
	}
	// Store the initial timestamp
	initialUpdatedAt := order.UpdatedAt

	// Act
	// We wait for a moment to ensure the timestamp will be different
	time.Sleep(1 * time.Millisecond)
	order.ChangeStatus(domain.StatusPaid)

	// Assert
	assert.Equal(t, domain.StatusPaid, order.Status)      // Check if status is updated
	assert.NotEqual(t, initialUpdatedAt, order.UpdatedAt) // Check if UpdatedAt was modified
}
