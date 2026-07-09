package database

import (
	"database/sql"
	"log"

	"github.com/elokanugrah/go-event-driven/internal/config"
)

func NewConnection(cfg *config.Config) *sql.DB {
	dsn := cfg.DSN()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("FATAL: Could not prepare database connection: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("FATAL: Database is not reachable: %v", err)
	}

	log.Println("Database connection successful.")
	return db
}
