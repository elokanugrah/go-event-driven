# Go Event Driven: A Production-Ready Architectural Blueprint

![Event Driven Architecture Concept](docs/images/repo_marketing_cover.png)

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/elokanugrah/go-event-driven)](https://github.com/elokanugrah/go-event-driven)
[![Architecture: Clean + Event-Driven](https://img.shields.io/badge/Architecture-Clean%20%2B%20Event--Driven-blue)](https://github.com/elokanugrah/go-event-driven)


A showcase of **Event-Driven Architecture (EDA)** combined with **Clean Architecture** in Go. This project demonstrates how to build decoupled, resilient, and highly scalable microservices using asynchronous messaging, database transaction integrity, and concurrency controls.

---

## 💡 Why Event-Driven?

Traditional systems tightly couple components. If a client creates an order, the system might synchronously try to save the order, charge the credit card, send an email, and update analytics inside a single slow request-response cycle. If the email server is down, the entire purchase fails.

This project solves that by decoupling operations:
* **Decoupling Services**: The REST API service handles fast client checkout validation and commits the order. Downstream tasks (sending confirmations, inventory reconciliation, reporting) are handled by a separate background **Worker Service**.
* **Asynchronous Processing**: Heavy tasks don't block the API response. Checkout responses are delivered in milliseconds.
* **Resilience**: If the Worker Service goes down, messages are safely queued in RabbitMQ and processed automatically when the worker recovers.
* **Horizontal Scalability**: Workers can be scaled dynamically to handle spike event loads without affecting the API's responsiveness.

---

## 🛡️ Enterprise-Grade Reliability & Resilience

Most basic event-driven applications suffer from two critical flaws under production traffic: **Dual Writes** and **Race Conditions**. This blueprint solves both.

### 1. The Dual-Write Problem & The Transactional Outbox Pattern
**The Problem**: A service needs to update its database and send an event to RabbitMQ. If it updates the database but the network drops before sending the event, the downstream systems never know about the order. If it sends the event first but the database update fails, the customer gets a confirmation for a purchase that does not exist.
**The Solution**: We implement the **Transactional Outbox Pattern**. 
1. The API saves the order and inserts the event payload into a local `outbox_events` table **inside the same database transaction**. If the order fails, the outbox event rolls back.
2. A background process (**Outbox Publisher**) polls the outbox table, publishes pending messages to RabbitMQ, and updates their status to `processed` upon success.
3. This guarantees **At-Least-Once Delivery**—even during service crashes, database restarts, or broker disconnects, no event is ever lost.

### 2. Double-Selling Prevention (Pessimistic Concurrency Control)
**The Problem**: If 100 users try to buy the last single item in stock at the exact same millisecond, a standard validation check will approve all of them, causing negative inventory or overselling.
**The Solution**: We apply row-level database locking (`FOR UPDATE`) when checking stock levels. Concurrent purchase transactions must wait in line for the lock, guaranteeing that inventory changes are strictly ordered and validated.

---

## 🗺️ Architectural Workflow

### High-Level Event Flow
```mermaid
graph LR
    Client["🖥️ Client"] -->|"POST /api/v1/orders"| API["⚡ API Service"]
    API -->|"Atomic Transaction"| DB[("🗄️ PostgreSQL Database<br/>(Order + Outbox)")]
    OP["📦 Outbox Publisher"] -.->|"Poll Pending Events"| DB
    OP -->|"Publish Events"| MQ["📨 RabbitMQ Broker"]
    MQ -->|"Deliver Asynchronously"| Worker["⚙️ Worker Service"]

    style Client fill:#6366f1,stroke:#4f46e5,color:#fff
    style API fill:#10b981,stroke:#059669,color:#fff
    style DB fill:#f59e0b,stroke:#d97706,color:#fff
    style OP fill:#14b8a6,stroke:#0d9488,color:#fff
    style MQ fill:#ef4444,stroke:#dc2626,color:#fff
    style Worker fill:#8b5cf6,stroke:#7c3aed,color:#fff
```

### Detailed Execution Sequence
```mermaid
sequenceDiagram
    participant C as Client
    participant API as API Service
    participant DB as PostgreSQL Database
    participant OP as Outbox Publisher (API Relay)
    participant MQ as RabbitMQ Message Broker
    participant W as Worker Service

    C->>API: POST /api/v1/orders (Buy Product X)
    activate API

    Note over API,DB: Database Transaction Begins
    API->>DB: BEGIN
    API->>DB: SELECT Product X (FOR UPDATE - Acquire Lock)
    API->>DB: Validate stock and decrement inventory count
    API->>DB: INSERT INTO orders & order_items
    API->>DB: INSERT INTO outbox_events (orders.created)
    API->>DB: COMMIT
    Note over API,DB: Transaction Completed Cleanly

    API-->>C: 201 Created (Instant Success Response)
    deactivate API

    Note over OP,DB: Background relay loop (Every 200ms)
    OP->>DB: SELECT pending events (FOR UPDATE SKIP LOCKED)
    OP->>MQ: Publish "orders.created" event to queue
    OP->>DB: UPDATE status = 'processed'

    MQ->>W: Deliver event payload
    activate W
    Note over W: Process asynchronously<br/>(Confirm email, build invoice, analytics)
    W-->>MQ: Acknowledge Message (ACK)
    deactivate W
```

---

## 🛠️ Tech Stack & Tooling

* **Go (v1.23+)** – Compiled, typed runtime language.
* **Gin Gonic** – Minimalist, high-performance HTTP router.
* **PostgreSQL** – Relational database with transactional support.
* **RabbitMQ** – Enterprise message broker managing queues.
* **Docker & Docker Compose** – Containerized runtime architecture.
* **Testify & Mockery** – Automated testing assertions and mock generation.
* **migrate-cli** – Database schema migration control.

---

## 📂 Project Structure

```
go-event-driven/
├── cmd/
│   ├── api/                  # REST API Service Entrypoint
│   │   └── main.go
│   ├── seed/                 # Database Seeding Utility (Products)
│   │   └── main.go
│   └── worker/               # Asynchronous Background Worker Entrypoint
│       └── main.go
├── migration/                # Database migrations (*.up.sql, *.down.sql)
├── internal/                 # Application Core
│   ├── config/               # Environment Configuration Loader
│   ├── database/             # PostgreSQL raw connection manager
│   ├── delivery/             # HTTP delivery transport layer (Gin)
│   ├── domain/               # Domain Models, business entities & pure state rules
│   ├── dto/                  # Data Transfer Objects (transport -> usecase boundaries)
│   ├── messagebroker/        # RabbitMQ publisher implementation
│   ├── repository/           # PostgreSQL repositories & Outbox repository
│   └── usecase/              # Business logic usecases & background OutboxPublisher
├── Dockerfile                # API Service container configuration
├── Dockerfile.worker         # Worker Service container configuration
└── docker-compose.yml        # Multi-service composition (database, broker, api, worker)
```

---

## 🚀 Getting Started

### Prerequisites
* [Go (v1.23+)](https://go.dev/)
* [Docker & Docker Compose](https://www.docker.com/)

### Installation & Execution

1.  **Clone the repository**
    ```bash
    git clone https://github.com/elokanugrah/go-event-driven.git
    cd go-event-driven
    ```

2.  **Create `.env` file**
    Create a `.env` file in the root directory:
    ```ini
    SERVER_PORT=9000
    DB_HOST=localhost
    DB_PORT=5432
    DB_USER=user
    DB_PASSWORD=password
    DB_NAME=order_db
    RABBITMQ_URL=amqp://guest:guest@localhost:5672/
    ```

3.  **Spin up the system containers**
    This spins up Postgres, RabbitMQ, the API Server, and the Background Worker:
    ```bash
    docker-compose up --build
    ```

4.  **Run migrations & seed initial products** (In a new terminal)
    ```bash
    # Install migrate CLI if you don't have it:
    go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

    # Apply database schemas (including orders, products, and outbox tables)
    migrate -database "postgres://user:password@localhost:5432/order_db?sslmode=disable" -path migration up

    # Seed demo product items
    go run ./cmd/seed
    ```

The API service is now listening at `http://localhost:9000`.

---

## 🔌 API Endpoints

### Products
| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `POST` | `/api/v1/products` | Create a new product |
| `GET` | `/api/v1/products` | Paginated list of products |
| `GET` | `/api/v1/products/{id}` | Get product details by ID |
| `PUT` | `/api/v1/products/{id}` | Update product attributes |
| `DELETE`| `/api/v1/products/{id}` | Delete a product |

### Orders
| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `POST` | `/api/v1/orders` | Creates a new order transactionally and queues an outbox event |

**Test Order Creation Request**:
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

---

## 🧪 Testing the Codebase

Unit and orchestration tests use Mockery-generated mocks to test layers in isolation. Run the test suites via:

```bash
# Regenerate mocks
go generate ./...

# Execute test suite
go test -v ./...
```