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

type ProductRepositorySuite struct {
	suite.Suite

	db   *sql.DB
	repo *postgres.PostgresProductRepository
}

// SetupSuite runs once before all tests in this suite.
// It's used for setting up the database connection.
func (s *ProductRepositorySuite) SetupSuite() {
	cfg := config.Load()
	s.db = database.NewConnection(cfg)
	s.repo = postgres.NewProductRepository(s.db)
}

// TearDownSuite runs once after all tests in this suite are finished.
func (s *ProductRepositorySuite) TearDownSuite() {
	err := s.db.Close()
	if err != nil {
		log.Fatalf("Failed to close test database connection: %v", err)
	}
}

// TearDownTest runs after each test function in the suite.
// It cleans all relevant tables to ensure test isolation.
func (s *ProductRepositorySuite) TearDownTest() {
	_, err := s.db.Exec("TRUNCATE TABLE products RESTART IDENTITY CASCADE")
	s.Suite.NoError(err)
}

// This function is the entry point for running the test suite.
func TestProductRepository(t *testing.T) {
	suite.Run(t, new(ProductRepositorySuite))
}

// TestSaveAndFindByID tests both Save and FindByID in a single flow.
func (s *ProductRepositorySuite) TestSaveAndFindByID() {
	assert := s.Suite.Assert()
	ctx := context.Background()

	newProduct := &domain.Product{
		Name:     "Kopi Arabica",
		Price:    120000,
		Quantity: 50,
	}

	err := s.repo.Save(ctx, newProduct)

	// Assert 1: Check for errors and that the ID is now populated
	assert.NoError(err)
	assert.NotZero(newProduct.ID) // The ID should be populated by the DB

	// Find the product we just saved using its new ID
	foundProduct, err := s.repo.FindByID(ctx, newProduct.ID)

	// Assert 2: Check for errors and that the found data matches
	assert.NoError(err)
	assert.NotNil(foundProduct)
	assert.Equal("Kopi Arabica", foundProduct.Name)
	assert.Equal(50, foundProduct.Quantity)
}

// TestFindByID_NotFound tests the case where a product ID does not exist.
func (s *ProductRepositorySuite) TestFindByID_NotFound() {
	assert := s.Suite.Assert()
	ctx := context.Background()

	// Act
	product, err := s.repo.FindByID(ctx, 99999)

	// Assert: We expect no SQL error, but the returned product should be nil
	assert.NoError(err)
	assert.Nil(product)
}

// TestUpdate tests if a product's data can be successfully updated.
func (s *ProductRepositorySuite) TestUpdate() {
	assert := s.Suite.Assert()
	ctx := context.Background()

	productToUpdate := &domain.Product{Name: "Buku Lama", Price: 50000, Quantity: 5}
	err := s.repo.Save(ctx, productToUpdate)
	assert.NoError(err)

	// Act
	productToUpdate.Name = "Buku Baru Edisi Revisi"
	productToUpdate.Quantity = 3
	err = s.repo.Update(ctx, productToUpdate)
	assert.NoError(err)

	// Assert: Fetch the product again and verify its fields are updated
	updatedProduct, err := s.repo.FindByID(ctx, productToUpdate.ID)
	assert.NoError(err)
	assert.NotNil(updatedProduct)
	assert.Equal("Buku Baru Edisi Revisi", updatedProduct.Name)
	assert.Equal(3, updatedProduct.Quantity)
}

// TestDelete tests if a product can be successfully removed.
func (s *ProductRepositorySuite) TestDelete() {
	assert := s.Suite.Assert()
	ctx := context.Background()

	productToDelete := &domain.Product{Name: "Barang Hapus", Price: 10, Quantity: 1}
	err := s.repo.Save(ctx, productToDelete)
	assert.NoError(err)

	// Act
	err = s.repo.Delete(ctx, productToDelete.ID)
	assert.NoError(err)

	// Assert: Try to find the deleted product, it should be nil
	foundProduct, err := s.repo.FindByID(ctx, productToDelete.ID)
	assert.NoError(err)
	assert.Nil(foundProduct)
}
