# AP2 Assignment 2 - gRPC Migration & Contract-First Development


## Repository Links

| Repository | Purpose | URL |
|---|---|---|
| **Proto Repository** | Source `.proto` files | `https://github.com/aknur111/my-user-service-protos` |
| **Generated Code Repository** | Auto-generated `.pb.go` files (v1.0.0) | `https://github.com/aknur111/my-user-service-gen` |


## What Changed from Assignment 1

| Component | Assignment 1 | Assignment 2 |
|---|---|---|
| Order -> Payment call | HTTP REST (`POST /payments`) | **gRPC** (`ProcessPayment` RPC) |
| `client/payment_client.go` | `http.Client` | `GRPCPaymentClient` (same interface) |
| Payment Service transport | HTTP only | HTTP + **gRPC Server** on `:50051` |
| Order Service transport | HTTP only | HTTP + **gRPC Streaming Server** on `:50052` |
| Domain / Use Cases | Unchanged | Unchanged |
| Repository layer | Unchanged | Unchanged |

**Clean Architecture is fully preserved** - only the Delivery layer was updated.


## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│  External User / Frontend                                           │
│           │  REST HTTP                                              │
│           ▼                                                         │
│  ┌────────────────────┐          ┌────────────────────────────┐    │
│  │   Order Service    │          │    Payment Service         │    │
│  │                    │          │                            │    │
│  │  HTTP :8080        │          │  HTTP  :8081 (backward)    │    │
│  │  ┌──────────────┐  │  gRPC    │  gRPC  :50051              │    │
│  │  │ HTTP Handler │  │ ──────►  │  ┌─────────────────────┐  │    │
│  │  └──────┬───────┘  │          │  │ gRPC Handler        │  │    │
│  │         │          │          │  │ + LoggingInterceptor│  │    │
│  │  ┌──────▼───────┐  │          │  └──────────┬──────────┘  │    │
│  │  │  Use Case    │  │          │             │              │    │
│  │  └──────┬───────┘  │          │  ┌──────────▼──────────┐  │    │
│  │         │          │          │  │    Use Case         │  │    │
│  │  ┌──────▼───────┐  │          │  └──────────┬──────────┘  │    │
│  │  │  Repository  │  │          │             │              │    │
│  │  └──────┬───────┘  │          │  ┌──────────▼──────────┐  │    │
│  │         │          │          │  │    Repository       │  │    │
│  │  ┌──────▼───────┐  │          │  └──────────┬──────────┘  │    │
│  │  │  order-db    │  │          │             │              │    │
│  │  │  (postgres)  │  │          │  ┌──────────▼──────────┐  │    │
│  │  └──────────────┘  │          │  │  payment-db         │  │    │
│  │                    │          │  │  (postgres)         │  │    │
│  │  gRPC :50052       │          │  └─────────────────────┘  │    │
│  │  ┌──────────────┐  │          └────────────────────────────┘    │
│  │  │ OrderStreaming│ │◄── gRPC streaming client (grpcurl/custom)  │
│  │  │ Handler      │  │                                            │
│  │  │ (polls DB)   │  │                                            │
│  │  └──────────────┘  │                                            │
│  └────────────────────┘                                            │
└─────────────────────────────────────────────────────────────────────┘

Contract-First Flow:
  [my-user-service-protos] ──GitHub Actions──► [my-user-service-gen]
                                                       │
                                             go get github.com/aknur111/my-user-service-gen@v1.2.0
                                                       │
                                          order-service & payment-service
```

![Architecture Diagram](image/Screenshot%202026-04-10%20at%2009.26.48.png)

## Proto Contract

### `service/payment/v1/payment.proto`
```protobuf
service PaymentService {
  rpc ProcessPayment(PaymentRequest) returns (PaymentResponse);
}
```

### `service/order/v1/order.proto`
```protobuf
service OrderService {
  rpc SubscribeToOrderUpdates(OrderRequest) returns (stream OrderStatusUpdate);
}
```


## How to Run

### Prerequisites
- Docker & Docker Compose
- Go 1.23+
- `grpcurl` (optional, for manual testing)

### 1. Start with Docker Compose
```bash
docker compose up --build
```

This starts:
- `order-db` on `localhost:5432`
- `payment-db` on `localhost:5433`
- `payment-service` on `localhost:8081` (HTTP) and `localhost:50051` (gRPC)
- `order-service` on `localhost:8080` (HTTP) and `localhost:50052` (gRPC)

### 2. Run locally (development)
```bash
# Terminal 1 — Payment Service
cd payment-service
cp .env.example .env
go run ./cmd/payment-service

# Terminal 2 — Order Service
cd order-service
cp .env.example .env
go run ./cmd/order-service
```


## Testing the gRPC Migration

### Create an order (triggers gRPC call to Payment Service)
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"cust-1","item_name":"Laptop","amount":50000}'
```

Expected response includes `"status":"Paid"` — the order service called Payment Service via gRPC internally.

### Subscribe to real-time order updates (Server-side Streaming)
```bash
# Install grpcurl: https://github.com/fullstorydev/grpcurl
grpcurl -plaintext \
  -d '{"order_id":"<ORDER_ID_HERE>"}' \
  localhost:50052 \
  service.order.v1.OrderService/SubscribeToOrderUpdates
```

While the stream is open, update the order status via REST in another terminal:
```bash
curl -X PATCH http://localhost:8080/orders/<ORDER_ID>/cancel
```

The stream will immediately push a new `OrderStatusUpdate` message with `status: "Cancelled"`.

### Verify Payment Service gRPC Interceptor logs
```bash
docker logs payment-service
# Expected output:
# {"level":"INFO","msg":"gRPC request started","method":"/service.payment.v1.PaymentService/ProcessPayment"}
# {"level":"INFO","msg":"gRPC request completed","method":"/service.payment.v1.PaymentService/ProcessPayment","duration_ms":3}
```


## Environment Variables Reference

### Order Service (`.env`)
| Variable | Default | Description |
|---|---|---|
| `HTTP_ADDR` | `:8080` | REST API listen address |
| `GRPC_ADDR` | `:50052` | gRPC streaming server address |
| `DB_DSN` | — | PostgreSQL connection string |
| `PAYMENT_GRPC_ADDR` | — | Payment Service gRPC address (e.g. `localhost:50051`) |
| `HTTP_TIMEOUT_SECONDS` | `5` | HTTP client timeout |
| `STREAM_POLL_INTERVAL_MS` | `500` | DB poll interval for order status streaming (ms) |

### Payment Service (`.env`)
| Variable | Default | Description |
|---|---|---|
| `HTTP_ADDR` | `:8081` | REST API listen address |
| `GRPC_ADDR` | `:50051` | gRPC server address |
| `DB_DSN` | — | PostgreSQL connection string |

