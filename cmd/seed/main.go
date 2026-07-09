package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/elokanugrah/go-event-driven/internal/config"
	"github.com/elokanugrah/go-event-driven/internal/database"

	"github.com/go-faker/faker/v4"
	_ "github.com/lib/pq" // PostgreSQL driver
)

func main() {
	cfg := config.Load()

	db := database.NewConnection(cfg)
	defer db.Close()

	if err := seedProducts(db); err != nil {
		log.Fatalf("FATAL: Failed to seed products: %v", err)
	}

	log.Println("Seeding process completed successfully!")
}

// seedProducts clears the products table and inserts new dummy data.
func seedProducts(db *sql.DB) error {
	// Clear existing data in the products table to make the script idempotent.
	log.Println("Clearing products table...")
	_, err := db.Exec(`TRUNCATE TABLE products RESTART IDENTITY CASCADE`)
	if err != nil {
		return fmt.Errorf("error truncating products table: %w", err)
	}

	stmt, err := db.Prepare(`INSERT INTO products (name, price, quantity, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`)
	if err != nil {
		return fmt.Errorf("error preparing insert statement: %w", err)
	}
	defer stmt.Close()

	log.Println("Inserting 50 dummy products...")

	// Insert 50 new dummy products in a single transaction for performance.
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
	}

	for i := 0; i < 50; i++ {
		// Generate fake data
		name := fmt.Sprintf("%s %s", faker.Word(), faker.Word()) // e.g., "Awesome Gadget"
		price := float64(rand.Intn(1000000) + 5000)              // Price between 5,000 and 1,005,000
		quantity := rand.Intn(100) + 10                          // Quantity between 10 and 110
		now := time.Now()

		// Execute the prepared statement within the transaction
		if _, err := tx.Stmt(stmt).Exec(name, price, quantity, now, now); err != nil {
			// If any insert fails, roll back the entire transaction
			if rbErr := tx.Rollback(); rbErr != nil {
				return fmt.Errorf("error executing insert and rolling back transaction: %w, %w", err, rbErr)
			}
			return fmt.Errorf("error executing insert: %w", err)
		}
	}

	// Commit the transaction.
	log.Println("Committing transaction...")
	return tx.Commit()
}
