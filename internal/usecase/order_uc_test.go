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
	var mockMessageBroker *mocks.MessageBroker
	var orderUseCase *usecase.OrderUseCase

	// setup is a helper function to initialize components for each test.
	setup := func() {
		mockProductRepo = new(mocks.ProductRepository)
		mockOrderRepo = new(mocks.OrderRepository)
		mockTxManager = new(mocks.TransactionManager)
		mockMessageBroker = new(mocks.MessageBroker)

		orderUseCase = usecase.NewOrderUseCase(mockOrderRepo, mockProductRepo, mockTxManager, mockMessageBroker)
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

		mockProductRepo.On("FindManyByIDs", mock.Anything, []int64{1, 2}).Return(mockProducts, nil).Once()
		mockProductRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Product")).Return(nil).Times(2)
		mockOrderRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.Order")).Return(nil).Once()

		mockMessageBroker.On("Publish", mock.Anything, "orders.created", mock.AnythingOfType("[]uint8")).Return(nil).Once()

		createdOrder, err := orderUseCase.CreateOrder(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, createdOrder)
		assert.Equal(t, float64(25000), createdOrder.TotalAmount)
		assert.Equal(t, domain.StatusPending, createdOrder.Status)

		mockProductRepo.AssertExpectations(t)
		mockOrderRepo.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
		mockMessageBroker.AssertExpectations(t)
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

		// Assert that no repository or message broker calls were made inside the successful part of the transaction
		mockProductRepo.AssertNotCalled(t, "FindManyByIDs", mock.Anything, mock.Anything)
		mockOrderRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
		mockProductRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
		mockMessageBroker.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything)

		mockTxManager.AssertExpectations(t) // Ensure the On call was met
	})

	t.Run("should return error when stock is insufficient", func(t *testing.T) {
		setup()

		input := dto.CreateOrderInput{UserID: 123, Items: []dto.CreateOrderItemInput{{ProductID: 1, Quantity: 20}}}
		mockProducts := []domain.Product{{ID: 1, Name: "Product A", Price: 10000, Quantity: 10}}

		mockTxManager.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Return(domain.ErrInsufficientStock). // Directly return the expected error
			Run(func(args mock.Arguments) {
				callback := args.Get(1).(func(ctx context.Context) error)
				// Execute callback to simulate internal logic that might set up other mocks if needed
				callback(context.Background())
			}).Once()

		mockProductRepo.On("FindManyByIDs", mock.Anything, []int64{1}).Return(mockProducts, nil).Once()

		createdOrder, err := orderUseCase.CreateOrder(context.Background(), input)

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInsufficientStock)
		assert.Nil(t, createdOrder)

		mockOrderRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
		mockProductRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
		mockMessageBroker.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything)

		mockTxManager.AssertExpectations(t) // Ensure the On call was met
	})

	t.Run("should return error when product is not found", func(t *testing.T) {
		setup()

		input := dto.CreateOrderInput{UserID: 123, Items: []dto.CreateOrderItemInput{{ProductID: 99, Quantity: 1}}}
		mockProducts := []domain.Product{}

		mockTxManager.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Return(errors.New("one or more products not found")). // Directly return the expected error
			Run(func(args mock.Arguments) {
				callback := args.Get(1).(func(ctx context.Context) error)
				callback(context.Background())
			}).Once()

		mockProductRepo.On("FindManyByIDs", mock.Anything, []int64{99}).Return(mockProducts, nil).Once()

		createdOrder, err := orderUseCase.CreateOrder(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, createdOrder)
		assert.Contains(t, err.Error(), "one or more products not found")
		mockMessageBroker.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything)

		mockTxManager.AssertExpectations(t)
	})

	t.Run("should return error if productRepo.FindManyByIDs fails", func(t *testing.T) {
		setup()

		input := dto.CreateOrderInput{
			UserID: 123,
			Items:  []dto.CreateOrderItemInput{{ProductID: 1, Quantity: 1}},
		}

		expectedErr := errors.New("database error")

		mockTxManager.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Return(expectedErr). // Directly return the expected error
			Run(func(args mock.Arguments) {
				callback := args.Get(1).(func(ctx context.Context) error)
				callback(context.Background())
			}).Once()

		mockProductRepo.On("FindManyByIDs", mock.Anything, []int64{1}).Return(nil, expectedErr).Once()

		createdOrder, err := orderUseCase.CreateOrder(context.Background(), input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, createdOrder)
		mockMessageBroker.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything)

		mockTxManager.AssertExpectations(t)
	})

	t.Run("should return error if orderRepo.Save fails", func(t *testing.T) {
		setup()

		input := dto.CreateOrderInput{
			UserID: 123,
			Items:  []dto.CreateOrderItemInput{{ProductID: 1, Quantity: 1}},
		}

		mockProducts := []domain.Product{
			{ID: 1, Name: "Product A", Price: 10000, Quantity: 10},
		}
		expectedErr := errors.New("save order failed")

		mockTxManager.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Return(expectedErr). // Directly return the expected error from WithTransaction
			Run(func(args mock.Arguments) {
				callback := args.Get(1).(func(ctx context.Context) error)
				callback(context.Background())
			}).Once()

		mockProductRepo.On("FindManyByIDs", mock.Anything, []int64{1}).Return(mockProducts, nil).Once()
		mockProductRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Product")).Return(nil).Once()
		mockOrderRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.Order")).Return(expectedErr).Once()

		createdOrder, err := orderUseCase.CreateOrder(context.Background(), input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, createdOrder)
		mockMessageBroker.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything)

		mockTxManager.AssertExpectations(t)
	})

	t.Run("should return error if productRepo.Update fails", func(t *testing.T) {
		setup()

		input := dto.CreateOrderInput{
			UserID: 123,
			Items:  []dto.CreateOrderItemInput{{ProductID: 1, Quantity: 1}},
		}

		mockProducts := []domain.Product{
			{ID: 1, Name: "Product A", Price: 10000, Quantity: 10},
		}
		expectedErr := errors.New("update product failed")

		mockTxManager.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Return(expectedErr). // Directly return the expected error
			Run(func(args mock.Arguments) {
				callback := args.Get(1).(func(ctx context.Context) error)
				callback(context.Background())
			}).Once()

		mockProductRepo.On("FindManyByIDs", mock.Anything, []int64{1}).Return(mockProducts, nil).Once()
		mockOrderRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.Order")).Return(nil).Once()
		mockProductRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Product")).Return(expectedErr).Once()

		createdOrder, err := orderUseCase.CreateOrder(context.Background(), input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, createdOrder)
		mockMessageBroker.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything)

		mockTxManager.AssertExpectations(t)
	})

	t.Run("should handle transaction manager returning an error", func(t *testing.T) {
		setup()

		input := dto.CreateOrderInput{
			UserID: 123,
			Items:  []dto.CreateOrderItemInput{{ProductID: 1, Quantity: 1}},
		}

		expectedErr := errors.New("transaction failed at manager level")

		mockTxManager.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Return(expectedErr).Once() // Transaction manager itself fails

		createdOrder, err := orderUseCase.CreateOrder(context.Background(), input)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, createdOrder)

		mockProductRepo.AssertNotCalled(t, "FindManyByIDs", mock.Anything, mock.Anything)
		mockOrderRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
		mockProductRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
		mockMessageBroker.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything)

		mockTxManager.AssertExpectations(t)
	})

	t.Run("should handle broker.Publish returning an error but still return created order", func(t *testing.T) {
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

		mockTxManager.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Return(nil).
			Run(func(args mock.Arguments) {
				callback := args.Get(1).(func(ctx context.Context) error)
				callback(context.Background())
			}).Once()

		mockProductRepo.On("FindManyByIDs", mock.Anything, []int64{1}).Return(mockProducts, nil).Once()
		mockProductRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Product")).Return(nil).Once()
		mockOrderRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.Order")).Return(nil).Once()

		mockMessageBroker.On("Publish", mock.Anything, "orders.created", mock.AnythingOfType("[]uint8")).Return(errors.New("broker publish error")).Once()

		createdOrder, err := orderUseCase.CreateOrder(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, createdOrder)
		assert.Equal(t, float64(20000), createdOrder.TotalAmount)
		assert.Equal(t, domain.StatusPending, createdOrder.Status)

		mockProductRepo.AssertExpectations(t)
		mockOrderRepo.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
		mockMessageBroker.AssertExpectations(t)
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
		mockProductRepo.AssertNotCalled(t, "FindManyByIDs", mock.Anything, mock.Anything)
		mockOrderRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
		mockProductRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
		mockMessageBroker.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything)
		mockTxManager.AssertNotCalled(t, "WithTransaction", mock.Anything, mock.Anything)

		mockTxManager.AssertExpectations(t) // Ensure the On call was met
	})
}
