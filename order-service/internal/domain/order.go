package domain

import "time"

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "Pending"
	OrderStatusPaid      OrderStatus = "Paid"
	OrderStatusFailed    OrderStatus = "Failed"
	OrderStatusCancelled OrderStatus = "Cancelled"
)

type Order struct {
	ID             string      `json:"id"`
	CustomerID     string      `json:"customer_id"`
	ItemName       string      `json:"item_name"`
	Amount         int64       `json:"amount"`
	Status         OrderStatus `json:"status"`
	CreatedAt      time.Time   `json:"created_at"`
	IdempotencyKey string      `json:"idempotency_key,omitempty"`
}