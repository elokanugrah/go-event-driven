package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/elokanugrah/go-event-driven/internal/domain"
	"github.com/elokanugrah/go-event-driven/internal/dto"
	"github.com/elokanugrah/go-event-driven/internal/usecase"
	"github.com/elokanugrah/go-event-driven/internal/usecase/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestOrderUseCase_CreateOrder(t *testing.T) {
	var mockProductRepo *mocks.ProductRepository
	var mockOrderRepo *mocks.OrderRepository
	var mockTxManager *mocks.TransactionManager
	var mockOutboxRepo *mocks.OutboxRepository
	var orderUseCase *usecase.OrderUseCase

	// setup is a helper function to initialize components for each test.
	setup := func() {
		mockProductRepo = new(mocks.ProductRepository)
		mockOrderRepo = new(mocks.OrderRepository)
		mockTxManager = new(mocks.TransactionManager)
		mockOutboxRepo = new(mocks.OutboxRepository)

		orderUseCase = usecase.NewOrderUseCase(mockOrderRepo, mockProductRepo, mockTxManager, mockOutboxRepo)
	}

	t.Run("should create order successfully when all conditions are met", func(t *testing.T) {
		setup()

		input := dto.CreateOrderInput{
			UserID: 123,
			Items: []dto.CreateOrderItemInput{
				{ProductID: 1, Quantity: 2},
				{ProductID: 2, Quantity: 1},
			},
		}

		mockProducts := []domain.Product{
			{ID: 1, Name: "Product A", Price: 10000, Quantity: 10},
			{ID: 2, Name: "Product B", Price: 5000, Quantity: 5},
		}

		mockTxManager.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Return(nil).
			Run(func(args mock.Arguments) {
				callback := args.Get(1).(func(ctx context.Context) error)
				callback(context.Background())
			}).Once()

		mockProductRepo.On("FindManyByIDsForUpdate", mock.Anything, []int64{1, 2}).Return(mockProducts, nil).Once()
		mockProductRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Product")).Return(nil).Times(2)
		mockOrderRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.Order")).Return(nil).Once()
		mockOutboxRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Once()

		createdOrder, err := orderUseCase.CreateOrder(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, createdOrder)
		assert.Equal(t, float64(25000), createdOrder.TotalAmount)
		assert.Equal(t, domain.StatusPending, createdOrder.Status)

		mockProductRepo.AssertExpectations(t)
		mockOrderRepo.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
		mockOutboxRepo.AssertExpectations(t)
	})

	t.Run("should return error when item quantity is not positive", func(t *testing.T) {
		setup()

		input := dto.CreateOrderInput{
			UserID: 123,
			Items: []dto.CreateOrderItemInput{
				{ProductID: 1, Quantity: 0}, // Invalid quantity
			},
		}

		// Arrange: Set up the TransactionManager mock to return the expected error.
		// This accounts for the possibility that the quantity check happens within the transaction logic.
		mockTxManager.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Return(errors.New("item quantity must be positive")).Once() // Simulate error from within transaction

		createdOrder, err := orderUseCase.CreateOrder(context.Background(), input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "item quantity must be positive")
		assert.Nil(t, createdOrder)

		// Assert that no repository or outbox calls were made inside the successful part of the transaction
		mockProductRepo.AssertNotCalled(t, "FindManyByIDsForUpdate", mock.Anything, mock.Anything)
		mockOrderRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
		mockProductRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
		mockOutboxRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)

		mockTxManager.AssertExpectations(t) // Ensure the On call was met
	})

	t.Run("should return error when no items in input", func(t *testing.T) {
		setup()

		input := dto.CreateOrderInput{
			UserID: 123,
			Items:  nil,
		}

		createdOrder, err := orderUseCase.CreateOrder(context.Background(), input)

		assert.Error(t, err)
		assert.Equal(t, "order must contain at least one item", err.Error())
		assert.Nil(t, createdOrder)

		mockProductRepo.AssertNotCalled(t, "FindManyByIDsForUpdate", mock.Anything, mock.Anything)
		mockOrderRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
		mockProductRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
		mockOutboxRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
		mockTxManager.AssertNotCalled(t, "WithTransaction", mock.Anything, mock.Anything)
	})

	t.Run("should return error if product not found", func(t *testing.T) {
		setup()

		input := dto.CreateOrderInput{
			UserID: 123,
			Items: []dto.CreateOrderItemInput{
				{ProductID: 1, Quantity: 2},
			},
		}

		mockProducts := []domain.Product{} // No products returned

		mockTxManager.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Return(errors.New("one or more products not found")).
			Run(func(args mock.Arguments) {
				callback := args.Get(1).(func(ctx context.Context) error)
				_ = callback(context.Background())
			}).Once()

		mockProductRepo.On("FindManyByIDsForUpdate", mock.Anything, []int64{1}).Return(mockProducts, nil).Once()

		createdOrder, err := orderUseCase.CreateOrder(context.Background(), input)

		assert.Error(t, err)
		assert.Equal(t, "one or more products not found", err.Error())
		assert.Nil(t, createdOrder)

		mockProductRepo.AssertExpectations(t)
		mockOrderRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
		mockProductRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
		mockOutboxRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
		mockTxManager.AssertExpectations(t)
	})

	t.Run("should return error if productRepo.FindManyByIDsForUpdate fails", func(t *testing.T) {
		setup()

		input := dto.CreateOrderInput{
			UserID: 123,
			Items: []dto.CreateOrderItemInput{
				{ProductID: 1, Quantity: 2},
			},
		}

		expectedErr := errors.New("database error")

		mockTxManager.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Return(expectedErr).
			Run(func(args mock.Arguments) {
				callback := args.Get(1).(func(ctx context.Context) error)
				_ = callback(context.Background())
			}).Once()

		mockProductRepo.On("FindManyByIDsForUpdate", mock.Anything, []int64{1}).Return(nil, expectedErr).Once()

		createdOrder, err := orderUseCase.CreateOrder(context.Background(), input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, createdOrder)

		mockProductRepo.AssertExpectations(t)
		mockOrderRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
		mockProductRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
		mockOutboxRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
		mockTxManager.AssertExpectations(t)
	})

	t.Run("should return error if stock is insufficient", func(t *testing.T) {
		setup()

		input := dto.CreateOrderInput{
			UserID: 123,
			Items: []dto.CreateOrderItemInput{
				{ProductID: 1, Quantity: 5},
			},
		}

		mockProducts := []domain.Product{
			{ID: 1, Name: "Product A", Price: 10000, Quantity: 2}, // Only 2 in stock, requested 5
		}

		mockTxManager.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Return(domain.ErrInsufficientStock).
			Run(func(args mock.Arguments) {
				callback := args.Get(1).(func(ctx context.Context) error)
				_ = callback(context.Background())
			}).Once()

		mockProductRepo.On("FindManyByIDsForUpdate", mock.Anything, []int64{1}).Return(mockProducts, nil).Once()

		createdOrder, err := orderUseCase.CreateOrder(context.Background(), input)

		assert.Error(t, err)
		assert.Equal(t, domain.ErrInsufficientStock, err)
		assert.Nil(t, createdOrder)

		mockProductRepo.AssertExpectations(t)
		mockOrderRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
		mockProductRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
		mockOutboxRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
		mockTxManager.AssertExpectations(t)
	})

	t.Run("should return error if orderRepo.Save fails", func(t *testing.T) {
		setup()

		input := dto.CreateOrderInput{
			UserID: 123,
			Items: []dto.CreateOrderItemInput{
				{ProductID: 1, Quantity: 2},
			},
		}

		mockProducts := []domain.Product{
			{ID: 1, Name: "Product A", Price: 10000, Quantity: 10},
		}

		expectedErr := errors.New("failed to save order")

		mockTxManager.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Return(expectedErr).
			Run(func(args mock.Arguments) {
				callback := args.Get(1).(func(ctx context.Context) error)
				_ = callback(context.Background())
			}).Once()

		mockProductRepo.On("FindManyByIDsForUpdate", mock.Anything, []int64{1}).Return(mockProducts, nil).Once()
		mockOrderRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.Order")).Return(expectedErr).Once()

		createdOrder, err := orderUseCase.CreateOrder(context.Background(), input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, createdOrder)

		mockProductRepo.AssertExpectations(t)
		mockOrderRepo.AssertExpectations(t)
		mockOutboxRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
		mockTxManager.AssertExpectations(t)
	})

	t.Run("should return error if outboxRepo.Save fails", func(t *testing.T) {
		setup()

		input := dto.CreateOrderInput{
			UserID: 123,
			Items: []dto.CreateOrderItemInput{
				{ProductID: 1, Quantity: 2},
			},
		}

		mockProducts := []domain.Product{
			{ID: 1, Name: "Product A", Price: 10000, Quantity: 10},
		}

		expectedErr := errors.New("outbox save error")

		mockTxManager.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Return(expectedErr).
			Run(func(args mock.Arguments) {
				callback := args.Get(1).(func(ctx context.Context) error)
				_ = callback(context.Background())
			}).Once()

		mockProductRepo.On("FindManyByIDsForUpdate", mock.Anything, []int64{1}).Return(mockProducts, nil).Once()
		mockProductRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Product")).Return(nil).Once()
		mockOrderRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.Order")).Return(nil).Once()
		mockOutboxRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.OutboxEvent")).Return(expectedErr).Once()

		createdOrder, err := orderUseCase.CreateOrder(context.Background(), input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, createdOrder)

		mockProductRepo.AssertExpectations(t)
		mockOrderRepo.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
		mockOutboxRepo.AssertExpectations(t)
	})

	t.Run("should return error when input items is empty", func(t *testing.T) {
		setup()

		input := dto.CreateOrderInput{
			UserID: 123,
			Items:  []dto.CreateOrderItemInput{}, // Empty items
		}

		createdOrder, err := orderUseCase.CreateOrder(context.Background(), input)

		assert.Error(t, err)
		assert.Equal(t, "order must contain at least one item", err.Error())
		assert.Nil(t, createdOrder)

		// Assert that no repository or message broker calls were made inside the transaction's successful path
		mockProductRepo.AssertNotCalled(t, "FindManyByIDsForUpdate", mock.Anything, mock.Anything)
		mockOrderRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
		mockProductRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
		mockOutboxRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
		mockTxManager.AssertNotCalled(t, "WithTransaction", mock.Anything, mock.Anything)

		mockTxManager.AssertExpectations(t) // Ensure the On call was met
	})
}
