package shared

import "time"

type Item struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
	Category    string  `json:"category"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

type ItemFilterInput struct {
	Category string `json:"category,omitempty"`
}

type OrderEvent struct {
	Type      string    `json:"type"`
	OrderID   string    `json:"orderId"`
	ItemID    string    `json:"itemId"`
	Quantity  int       `json:"quantity"`
	Timestamp time.Time `json:"timestamp"`
}

type AppSyncEvent struct {
	FieldName string                 `json:"fieldName"`
	Arguments map[string]interface{} `json:"arguments"`
	Identity  interface{}            `json:"identity"`
	Source    interface{}            `json:"source"`
	Request   interface{}            `json:"request"`
	Prev      interface{}            `json:"prev"`
}

type CreateItemInput struct {
	Sku         string  `json:"sku"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
	Category    string  `json:"category"`
}

type UpdateItemInput struct {
	ID          string  `json:"id"`
	Sku         string  `json:"sku,omitempty"`
	Name        string  `json:"name,omitempty"`
	Description string  `json:"description,omitempty"`
	Quantity    int     `json:"quantity,omitempty"`
	UnitPrice   float64 `json:"unitPrice,omitempty"`
	Category    string  `json:"category,omitempty"`
}
