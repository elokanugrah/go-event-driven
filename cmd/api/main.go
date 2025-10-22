package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/elokanugrah/go-order-system/internal/config"
	"github.com/elokanugrah/go-order-system/internal/database"
	"github.com/elokanugrah/go-order-system/internal/messagebroker"
	"github.com/elokanugrah/go-order-system/internal/repository/postgres"
	"github.com/elokanugrah/go-order-system/internal/usecase"

	httpDelivery "github.com/elokanugrah/go-order-system/internal/delivery/http"

	_ "github.com/lib/pq"
)

func main() {
	// Load configuration
	cfg := config.Load()

	db := database.NewConnection(cfg)
	defer db.Close()
	// --- WIRING / DEPENDENCY INJECTION ---

	// Initialize Message Broker
	mb, err := messagebroker.NewRabbitMQBroker(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to initialize message broker: %v", err)
	}

	// Initialize Repository Layer
	productRepo := postgres.NewProductRepository(db)
	orderRepo := postgres.NewOrderRepository(db)
	txManager := postgres.NewTransactionManager(db)

	// Initialize Usecase Layer
	productUseCase := usecase.NewProductUseCase(productRepo)
	orderUseCase := usecase.NewOrderUseCase(orderRepo, productRepo, txManager, mb)

	// Initialize Delivery Layer (Handler)
	apiHandler := httpDelivery.NewHandler(productUseCase, orderUseCase)

	// Setup Router
	router := httpDelivery.SetupRouter(apiHandler)

	// --- GRACEFUL SHUTDOWN SETUP ---
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	// Start server in a goroutine so that it doesn't block.
	go func() {
		log.Printf("Starting server on port %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout.
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need to add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
