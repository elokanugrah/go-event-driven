# Go Order System

[![Go Report Card](https://goreportcard.com/badge/github.com/elokanugrah/go-order-system)](https://goreportcard.com/report/github.com/elokanugrah/go-order-system)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A backend service for a simple order processing system, built with Go, demonstrating Clean Architecture and Event-Driven patterns.

The system features a REST API for managing products and orders, and a separate Worker Service that processes events asynchronously using RabbitMQ.

## Core Concepts

  * **Clean Architecture:** Separates concerns into distinct layers (Domain, Usecase, Repository, Delivery) for maintainability and testability.
  * **Event-Driven:** Uses RabbitMQ to decouple the order creation process from subsequent actions (e.g., sending notifications).
  * **Transactional Integrity:** Ensures that creating an order and updating product stock are atomic operations.
  * **Testing:** Includes both unit tests and integration tests.
  * **Containerized:** Fully containerized with Docker and orchestrated with Docker Compose for a consistent environment.

## Architecture Diagram
![Architecture Diagram](docs/images/architecture-diagram.png)

## Tech Stack

  * **Language:** Go
  * **Framework:** Gin Gonic
  * **Database:** PostgreSQL
  * **Message Broker:** RabbitMQ
  * **Containerization:** Docker
  * **Testing:** Testify, Mockery

## Getting Started

### Prerequisites

  * Go (v1.23+)
  * Docker & Docker Compose
  * `migrate-cli`
  * `mockery`

### Tooling Installation

  * `migrate-cli` is a command-line tool for managing database migrations.

    ```bash
    go install -tags 'postgres' [github.com/golang-migrate/migrate/v4/cmd/migrate@latest](https://github.com/golang-migrate/migrate/v4/cmd/migrate@latest)
    ```
  * `mockery` is a tool for automatically generating mocks for your Go interfaces.
  
    ```bash
    go install [github.com/vektra/mockery/v2@latest](https://github.com/vektra/mockery/v2@latest)
    ```

### Running with Docker

1.  **Clone the repository**

    ```bash
    git clone https://github.com/elokanugrah/go-order-system.git
    cd go-order-system
    ```

2.  **Create `.env` file**
    Create a `.env` file in the root directory.

    ```ini
    SERVER_PORT=9000
    DB_HOST=localhost
    DB_PORT=5432
    DB_USER=user
    DB_PASSWORD=password
    DB_NAME=order_db
    RABBITMQ_URL=amqp://guest:guest@localhost:5672/
    ```

3.  **Run Services**

    ```bash
    docker-compose up --build
    ```

4.  **Run Migrations & Seeder** (in a new terminal)

    ```bash
    # Create database schema
    migrate -database "postgres://user:password@localhost:5432/order_db?sslmode=disable" -path migration up

    # Seed product data
    go run ./cmd/seed
    ```

The API is now running at `http://localhost:9000`.

## API Endpoints

### Products

| Method | Endpoint              | Description              |
| :----- | :-------------------- | :----------------------- |
| `POST` | `/api/v1/products`      | Create a new product.    |
| `GET`  | `/api/v1/products`      | List all products.       |
| `GET`  | `/api/v1/products/{id}` | Get a product by its ID. |
| `PUT`  | `/api/v1/products/{id}` | Update a product.        |
| `DELETE`| `/api/v1/products/{id}` | Delete a product.        |

### Orders

| Method | Endpoint           | Description                                                        |
| :----- | :----------------- | :----------------------------------------------------------------- |
| `POST` | `/api/v1/orders`   | Creates a new order and publishes an event to RabbitMQ for the worker. |

**Example: Create an Order**

```bash
curl -X POST http://localhost:9000/api/v1/orders \
-H "Content-Type: application/json" \
-d '{
    "user_id": 123,
    "items": [
        {
            "product_id": 1,
            "quantity": 2
        }
    ]
}'
```

## Running Tests

To run all unit and integration tests, ensure the database is running and execute:

```bash
go test -v ./...
```