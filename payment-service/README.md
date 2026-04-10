
# Payment Service

Payment Service processes payment authorization requests for orders.

## Responsibilities

- Authorize or decline payments
- Store payment records in its own database
- Return payment status for a given order

## Clean Architecture

Project structure:

```text
payment-service/
├── cmd/
│   └── payment-service/
│       └── main.go
├── internal/
│   ├── app/
│   ├── domain/
│   ├── repository/
│   ├── transport/
│   │   └── http/
│   └── usecase/
├── migrations/
└── README.md
```

## Layers

### domain
Contains payment entity, statuses, and domain errors.

### usecase
Contains business rules:
- amount must be greater than 0
- if amount is greater than `100000`, payment is declined
- otherwise payment is authorized
- every payment receives a unique transaction ID

### repository
Contains PostgreSQL access logic.

### transport/http
Thin Gin handlers for request parsing and response formatting.

### cmd/main.go
Manual dependency wiring.


## Bounded Context

Payment Service owns only payment data.

It has:
- its own models
- its own repository
- its own database

Order Service cannot read or write the Payment database directly.


## Database

This service uses its own PostgreSQL database, for example:

- `payment_db`

Payments table

```
CREATE TABLE IF NOT EXISTS payments (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL UNIQUE,
    transaction_id TEXT NOT NULL,
    amount BIGINT NOT NULL,
    status TEXT NOT NULL
);
```
## Business Rules

- Money uses `int64`
- `amount > 0`
- if `amount > 100000`, payment status is `Declined`
- otherwise payment status is `Authorized`
- transaction ID is generated for each payment

---

## REST API

### 1. Create Payment

**POST** `/payments`

#### Request body

```json
{
  "order_id": "order-id",
  "amount": 50000
}

  ```

### Success response
```
{
  "id": "payment-id",
  "order_id": "order-id",
  "transaction_id": "transaction-id",
  "amount": 50000,
  "status": "Authorized"
}
```

If amount is too large
```
{
  "id": "payment-id",
  "order_id": "order-id",
  "transaction_id": "transaction-id",
  "amount": 200000,
  "status": "Declined"
}
```
## 2. Get Payment by Order ID

GET /payments/{order_id}
```
curl http://localhost:8081/payments/{order_id}
```

## Run Locally
### 1. Start PostgreSQL

Create database:
```
CREATE DATABASE payment_db;
```

### 2. Apply migration
```
psql -U postgres -d payment_db -f migrations/001_create_payments.sql
```
### 3. Run service
```
DB_DSN="postgres://postgres:YOUR_PASSWORD@localhost:5432/payment_db?sslmode=disable" \
PORT=8081 \
go run ./cmd/payment-service
```
### Example curl commands
### Authorized payment
```
curl -X POST http://localhost:8081/payments \
-H "Content-Type: application/json" \
-d '{"order_id":"order-1","amount":50000}'
```
### Declined payment
```
curl -X POST http://localhost:8081/payments \
-H "Content-Type: application/json" \
-d '{"order_id":"order-2","amount":200000}'
```
### Get payment
```
curl http://localhost:8081/payments/order-1
```
## Design Decisions
- Separate database per service
- No shared code with Order Service
- Thin handlers, logic in use case
- PostgreSQL used as real persistent storage
- Payment authorization exposed through REST

## Additional Features (Beyond Assignment Requirements)

This project includes several improvements beyond the base assignment to better simulate a production-ready microservices system.

---

### Idempotency Support (Order Service)

Order creation supports idempotent requests using the `Idempotency-Key` header.

- If the same key is sent multiple times, the same order is returned
- Prevents duplicate orders in case of retries or network failures
- Implemented at the use case and database level (unique constraint)



### Structured Logging

Both services use structured JSON logging (`log/slog`):

- Each request includes a unique `request_id`
- Logs include important fields like `order_id`, `status`, `error`
- Helps with debugging and tracing requests across services

---

### Middleware

Custom middleware is added:

- `RequestIDMiddleware` — assigns a unique ID to every request
- Logging middleware — logs request lifecycle
- Centralized error handling format

---

### Unified Error Responses

All errors follow a consistent JSON structure:

```json
{
  "error": "description",
  "code": "ERROR_CODE",
  "request_id": "uuid"
}
```
### Health & Readiness Endpoints

Each service exposes:
```
GET /health — basic liveness check
GET /ready — readiness check (for orchestration systems)
```

### Swagger (OpenAPI Documentation)

Both services include Swagger UI:

- Interactive API documentation
- Allows testing endpoints directly from browser
- Generated using swaggo

### Endpoints:
```
Order Service -> /swagger/index.html
Payment Service -> /swagger/index.html
```
### Graceful Shutdown

Services handle shutdown signals (SIGINT, SIGTERM):

- Complete in-flight requests before stopping
- Prevents abrupt termination
- Improves reliability

### HTTP Client Timeout

Order Service uses a custom HTTP client:

- Timeout is configurable (default: 2 seconds)
- Prevents hanging requests to Payment Service
- Ensures system responsiveness

### Unit Tests

Business logic is covered with unit tests:

- Order use case tests
- Payment use case tests
- Mocked repositories for isolation