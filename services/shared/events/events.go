package events

import "time"

type OrderCreatedEvent struct {
	OrderID   string    `json:"orderId"`
	ItemID    string    `json:"itemId"`
	Quantity  int       `json:"quantity"`
	Timestamp time.Time `json:"timestamp"`
}

type OrderCancelledEvent struct {
	OrderID   string    `json:"orderId"`
	ItemID    string    `json:"itemId"`
	Quantity  int       `json:"quantity"`
	Timestamp time.Time `json:"timestamp"`
}

const (
	EventTypeOrderCreated   = "ORDER_CREATED"
	EventTypeOrderCancelled = "ORDER_CANCELLED"
)
