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

func TestProductUseCase(t *testing.T) {
	var mockProductRepo *mocks.ProductRepository
	var productUseCase *usecase.ProductUseCase

	// setup is a helper function to reset mocks for each test group.
	setup := func() {
		mockProductRepo = new(mocks.ProductRepository)
		productUseCase = usecase.NewProductUseCase(mockProductRepo)
	}

	t.Run("GetProductByID", func(t *testing.T) {
		setup()
		t.Run("should return product successfully when product is found", func(t *testing.T) {
			expectedProduct := &domain.Product{ID: 1, Name: "Found Product"}
			mockProductRepo.On("FindByID", mock.Anything, int64(1)).Return(expectedProduct, nil).Once()

			product, err := productUseCase.GetProductByID(context.Background(), 1)

			// Assert
			assert.NoError(t, err)
			assert.NotNil(t, product)
			assert.Equal(t, int64(1), product.ID)
			mockProductRepo.AssertExpectations(t)
		})
	})

	t.Run("CreateProduct", func(t *testing.T) {
		setup()
		t.Run("should create product successfully with valid input", func(t *testing.T) {
			input := dto.CreateProductInput{Name: "New Gadget", Price: 1500, Quantity: 100}

			// When Save is called, we tell the mock to do nothing and return no error.
			// Use mock.MatchedBy to check if the argument passed to Save has the correct name.
			mockProductRepo.On("Save", mock.Anything, mock.MatchedBy(func(p *domain.Product) bool {
				return p.Name == input.Name
			})).Return(nil).Once()

			product, err := productUseCase.CreateProduct(context.Background(), input)

			// Assert
			assert.NoError(t, err)
			assert.NotNil(t, product)
			assert.Equal(t, "New Gadget", product.Name)
			mockProductRepo.AssertExpectations(t)
		})

		t.Run("should return error on invalid input", func(t *testing.T) {
			input := dto.CreateProductInput{Name: "", Price: 1500, Quantity: 100} // Empty name

			// don't set up the mock here because the function should fail before calling the repo.
			product, err := productUseCase.CreateProduct(context.Background(), input)

			// Assert
			assert.Error(t, err)
			assert.Nil(t, product)
		})
	})

	t.Run("ListProducts", func(t *testing.T) {
		setup()
		t.Run("should list products successfully", func(t *testing.T) {
			expectedProducts := []domain.Product{
				{ID: 1, Name: "Product 1"},
				{ID: 2, Name: "Product 2"},
			}
			limit, offset := 10, 0
			mockProductRepo.On("FindAll", mock.Anything, limit, offset).Return(expectedProducts, nil).Once()

			products, err := productUseCase.ListProducts(context.Background(), 1, 10)

			// Assert
			assert.NoError(t, err)
			assert.Len(t, products, 2)
			mockProductRepo.AssertExpectations(t)
		})
	})

	t.Run("UpdateProduct", func(t *testing.T) {
		t.Run("should update product successfully", func(t *testing.T) {
			setup()
			input := dto.UpdateProductInput{Name: "Updated Name", Price: 200, Quantity: 20}
			existingProduct := &domain.Product{ID: 1, Name: "Old Name", Price: 100, Quantity: 10}

			// Mock the two repository calls needed for an update.
			mockProductRepo.On("FindByID", mock.Anything, int64(1)).Return(existingProduct, nil).Once()
			mockProductRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Product")).Return(nil).Once()

			updatedProduct, err := productUseCase.UpdateProduct(context.Background(), 1, input)

			// Assert
			assert.NoError(t, err)
			assert.NotNil(t, updatedProduct)
			assert.Equal(t, "Updated Name", updatedProduct.Name)
			assert.Equal(t, 20, updatedProduct.Quantity)
			mockProductRepo.AssertExpectations(t)
		})

		t.Run("should return not found error when updating non-existent product", func(t *testing.T) {
			setup()
			input := dto.UpdateProductInput{Name: "Updated Name", Price: 200, Quantity: 20}

			// Mock FindByID to return "not found".
			mockProductRepo.On("FindByID", mock.Anything, int64(99)).Return(nil, nil).Once()

			product, err := productUseCase.UpdateProduct(context.Background(), 99, input)

			// Assert
			assert.Error(t, err)
			assert.True(t, errors.Is(err, usecase.ErrProductNotFound))
			assert.Nil(t, product)
			// Ensure Update was never called
			mockProductRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
		})
	})

	t.Run("DeleteProduct", func(t *testing.T) {
		setup()
		t.Run("should delete product successfully", func(t *testing.T) {
			existingProduct := &domain.Product{ID: 1}
			mockProductRepo.On("FindByID", mock.Anything, int64(1)).Return(existingProduct, nil).Once()
			mockProductRepo.On("Delete", mock.Anything, int64(1)).Return(nil).Once()

			err := productUseCase.DeleteProduct(context.Background(), 1)

			// Assert
			assert.NoError(t, err)
			mockProductRepo.AssertExpectations(t)
		})
	})
}
