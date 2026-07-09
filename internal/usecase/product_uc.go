package usecase

import (
	"context"
	"errors"

	"github.com/elokanugrah/go-event-driven/internal/domain"
	"github.com/elokanugrah/go-event-driven/internal/dto"
)

var ErrProductNotFound = errors.New("product not found")

type ProductUseCase struct {
	productRepo ProductRepository
}

func NewProductUseCase(pr ProductRepository) *ProductUseCase {
	return &ProductUseCase{
		productRepo: pr,
	}
}

// CreateProduct handles the logic for creating a new product.
func (uc *ProductUseCase) CreateProduct(ctx context.Context, input dto.CreateProductInput) (*domain.Product, error) {
	// Validate input data.
	if input.Name == "" {
		return nil, errors.New("product name cannot be empty")
	}
	if input.Price <= 0 {
		return nil, errors.New("product price must be positive")
	}
	if input.Quantity < 0 {
		return nil, errors.New("product quantity cannot be negative")
	}

	newProduct := &domain.Product{
		Name:     input.Name,
		Price:    input.Price,
		Quantity: input.Quantity,
	}

	err := uc.productRepo.Save(ctx, newProduct)
	if err != nil {
		return nil, err
	}

	return newProduct, nil
}

// GetProductByID contains the logic for retrieving a single product.
func (uc *ProductUseCase) GetProductByID(ctx context.Context, id int64) (*domain.Product, error) {
	product, err := uc.productRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}
	return product, nil
}

// ListProducts handles listing all products with pagination.
func (uc *ProductUseCase) ListProducts(ctx context.Context, page, pageSize int) ([]domain.Product, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 { // Limit page size to a max of 100.
		pageSize = 10
	}

	// Calculate offset for the database query.
	offset := (page - 1) * pageSize

	products, err := uc.productRepo.FindAll(ctx, pageSize, offset)
	if err != nil {
		return nil, err
	}

	return products, nil
}

// UpdateProduct handles the logic for updating an existing product.
func (uc *ProductUseCase) UpdateProduct(ctx context.Context, id int64, input dto.UpdateProductInput) (*domain.Product, error) {
	productToUpdate, err := uc.productRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if productToUpdate == nil {
		return nil, ErrProductNotFound
	}

	if input.Name == "" {
		return nil, errors.New("product name cannot be empty")
	}
	if input.Price <= 0 {
		return nil, errors.New("product price must be positive")
	}
	if input.Quantity < 0 {
		return nil, errors.New("product quantity cannot be negative")
	}

	// Update the fields of the existing domain object.
	productToUpdate.Name = input.Name
	productToUpdate.Price = input.Price
	productToUpdate.Quantity = input.Quantity

	err = uc.productRepo.Update(ctx, productToUpdate)
	if err != nil {
		return nil, err
	}

	return productToUpdate, nil
}

// DeleteProduct handles the logic for deleting a product.
func (uc *ProductUseCase) DeleteProduct(ctx context.Context, id int64) error {
	product, err := uc.productRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if product == nil {
		return ErrProductNotFound
	}

	return uc.productRepo.Delete(ctx, id)
}
