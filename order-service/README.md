# Order Service

Order Service manages customer orders and coordinates payment authorization by calling Payment Service over REST.

## Responsibilities

- Create a new order
- Get order by ID
- Cancel pending order
- Call Payment Service to authorize payment
- Update order status based on payment result
- Support idempotent order creation with `Idempotency-Key`

## Clean Architecture

Project structure:

```text
order-service/
├── cmd/
│   └── order-service/
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
Contains business entities, statuses, and domain errors.  
Domain models do not depend on HTTP or database details.

### usecase
Contains business logic:
- amount must be greater than 0
- order is first created as `Pending`
- after payment authorization it becomes `Paid` or `Failed`
- only `Pending` orders can be cancelled
- `Paid` orders cannot be cancelled
- repeated requests with the same `Idempotency-Key` return the existing order

### repository
Contains PostgreSQL persistence logic.

### transport/http
Thin Gin handlers:
- parse request
- call use case
- return HTTP response

### cmd/main.go
Composition root with manual dependency injection.


## Bounded Context

Order Service owns only order data.

- It does not access Payment Service database directly  
- It communicates with Payment Service only through REST


## Database

This service uses its own PostgreSQL database, for example:

- `order_db`

### Orders table
```
CREATE TABLE IF NOT EXISTS orders (
    id TEXT PRIMARY KEY,
    customer_id TEXT NOT NULL,
    item_name TEXT NOT NULL,
    amount BIGINT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    idempotency_key TEXT UNIQUE
);

```
## Business Rules

- Money is stored as `int64`
- `amount > 0`
- New order status starts as `Pending`
- If payment is authorized, order becomes `Paid`
- If payment is declined, order becomes `Failed`
- If Payment Service is unavailable, Order Service returns `503 Service Unavailable`
- In failure scenario, order is marked as `Failed`
- `Paid` orders cannot be cancelled
- Only `Pending` orders can be cancelled

---

## REST API

### 1. Create Order

**POST** `/orders`

#### Request body

```json
{
  "customer_id": "123",
  "item_name": "Book",
  "amount": 50000
}
```

#### Optional header
```
Idempotency-Key: test123

```

#### Success response
```
{
  "id": "order-id",
  "customer_id": "123",
  "item_name": "Book",
  "amount": 50000,
  "status": "Paid",
  "created_at": "2026-04-01T06:57:55Z"
}
```

### 2. Get Order by ID

GET /orders/{id}

```
curl http://localhost:8080/orders/{id}
```

### 3. Cancel Order
```
PATCH /orders/{id}/cancel
```
#### Rule

Only Pending orders can be cancelled.

### Interaction with Payment Service

####  Order Service calls:
```
POST http://localhost:8081/payments
```
A custom http.Client with timeout is used to avoid hanging when Payment Service is unavailable.

### Failure Handling

If Payment Service is down or times out:

- Order Service does not hang indefinitely
- it returns 503 Service Unavailable
- order status is updated to Failed

### Idempotency

Bonus task implementation:

- Client sends Idempotency-Key header
- Order Service checks whether an order with the same key already exists
- If it exists, the existing order is returned
- No duplicate order is created
- No duplicate payment request is sent

## Run Locally
### 1. Start PostgreSQL

Create database:
```
CREATE DATABASE order_db;
```

### 2. Apply migration
```
psql -U postgres -d order_db -f migrations/001_create_orders.sql
```
If the table already exists, add the bonus column:
```
ALTER TABLE orders ADD COLUMN IF NOT EXISTS idempotency_key TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS ux_orders_idempotency_key ON orders (idempotency_key);
```

### 3. Run service
```
DB_DSN="postgres://postgres:YOUR_PASSWORD@localhost:5432/order_db?sslmode=disable" \
PORT=8080 \
PAYMENT_BASE_URL="http://localhost:8081" \
go run ./cmd/order-service

```

### Example curl commands
#### Normal order
```
curl -X POST http://localhost:8080/orders \
-H "Content-Type: application/json" \
-d '{"customer_id":"123","item_name":"Book","amount":50000}'
```
### Invalid amount
```
curl -X POST http://localhost:8080/orders \
-H "Content-Type: application/json" \
-d '{"customer_id":"123","item_name":"Book","amount":0}'
```
### Payment declined
```
curl -X POST http://localhost:8080/orders \
-H "Content-Type: application/json" \
-d '{"customer_id":"123","item_name":"Laptop","amount":200000}'
```
### Idempotent request
```
curl -X POST http://localhost:8080/orders \
-H "Content-Type: application/json" \
-H "Idempotency-Key: test123" \
-d '{"customer_id":"1","item_name":"Test","amount":1000}'
```
Repeat the same request with the same key and the same existing order will be returned.

### Design Decisions
- Separate database per service

- No shared entity package between services
- Manual dependency injection in main.go
- Business logic isolated in use cases
- Repository interfaces used as ports
- REST used for synchronous inter-service communication
