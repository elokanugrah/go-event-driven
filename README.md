# Go Event Driven

[![Go Report Card](https://goreportcard.com/badge/github.com/elokanugrah/go-event-driven)](https://goreportcard.com/report/github.com/elokanugrah/go-event-driven)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A production-ready backend service showcasing **Event-Driven Architecture** (EDA) in Go. This project demonstrates how to build decoupled, scalable microservices using asynchronous messaging with RabbitMQ.

The system features a **REST API** for managing products and orders, and a separate **Worker Service** that reacts to domain events published to a message broker — processing them asynchronously and independently from the main API.

## Why Event-Driven?

Traditional request-response architectures tightly couple services together, making them harder to scale and maintain. This project solves that by:

  * **Decoupling Services:** The API service publishes events without knowing or caring who consumes them. The Worker service processes events independently.
  * **Asynchronous Processing:** Heavy tasks (e.g., sending notifications, updating analytics) don't block the API response.
  * **Scalability:** Workers can be scaled horizontally to handle increased event load without affecting the API.
  * **Resilience:** If a worker goes down, messages are persisted in RabbitMQ and processed when the worker recovers.

## Core Concepts

  * **Event-Driven Architecture:** Uses RabbitMQ as the central message broker to decouple the order creation process from downstream actions (e.g., sending notifications, updating reports).
  * **Clean Architecture:** Separates concerns into distinct layers (Domain, Usecase, Repository, Delivery) for maintainability and testability.
  * **Transactional Integrity:** Ensures that creating an order and updating product stock are atomic operations within a database transaction.
  * **Testing:** Includes both unit tests (with mocks) and integration tests (with a live database).
  * **Containerized:** Fully containerized with Docker and orchestrated with Docker Compose for a consistent development environment.

## Architecture Diagram
![Architecture Diagram](docs/images/architecture-diagram.png)

### Event Flow

```mermaid
graph LR
    C["🖥️ Client"] -->|"POST /api/v1/orders"| API["⚡ API Service"]
    API -->|"Transaction"| DB[("🗄️ PostgreSQL")]
    API -->|"Publish Event"| MQ["📨 RabbitMQ"]
    MQ -->|"Consume Event"| W["⚙️ Worker Service"]

    style C fill:#6366f1,stroke:#4f46e5,color:#fff
    style API fill:#10b981,stroke:#059669,color:#fff
    style DB fill:#f59e0b,stroke:#d97706,color:#fff
    style MQ fill:#ef4444,stroke:#dc2626,color:#fff
    style W fill:#8b5cf6,stroke:#7c3aed,color:#fff
```

> **How it works:** Client creates an order → API persists it in a **database transaction** → publishes an `orders.created` event to **RabbitMQ** → Worker picks up the event **asynchronously** and processes it (e.g., send confirmation email, update reports).

<details>
<summary><strong>📋 Detailed Sequence Diagram</strong></summary>

```mermaid
sequenceDiagram
    participant C as Client
    participant API as API Service
    participant DB as PostgreSQL
    participant MQ as RabbitMQ
    participant W as Worker Service

    C->>API: POST /api/v1/orders
    activate API

    Note over API,DB: Database Transaction
    API->>DB: BEGIN
    API->>DB: SELECT products (validate stock)
    API->>DB: INSERT INTO orders + order_items
    API->>DB: UPDATE products (decrease stock)
    API->>DB: COMMIT

    API->>MQ: Publish "orders.created" event
    API-->>C: 201 Created
    deactivate API

    MQ->>W: Deliver message
    activate W
    Note over W: Process event async<br/>(e.g. send email, update reports)
    W-->>MQ: ACK
    deactivate W
```

</details>

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
    go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
    ```
  * `mockery` is a tool for automatically generating mocks for your Go interfaces.
  
    ```bash
    go install github.com/vektra/mockery/v2@latest
    ```

### Running with Docker

1.  **Clone the repository**

    ```bash
    git clone https://github.com/elokanugrah/go-event-driven.git
    cd go-event-driven
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

## Project Structure

```
go-event-driven/
├── cmd/
│   ├── api/          # REST API entrypoint
│   ├── worker/       # Event consumer (RabbitMQ worker)
│   └── seed/         # Database seeder
├── internal/
│   ├── config/       # Environment-based configuration
│   ├── database/     # Database connection
│   ├── domain/       # Core business entities & rules
│   ├── dto/          # Data Transfer Objects
│   ├── usecase/      # Application business logic
│   ├── repository/   # Data persistence (PostgreSQL)
│   ├── delivery/     # HTTP handlers & routing (Gin)
│   └── messagebroker/# RabbitMQ publisher
├── migration/        # SQL schema migrations
├── docker-compose.yml
├── Dockerfile        # API service container
└── Dockerfile.worker # Worker service container
```