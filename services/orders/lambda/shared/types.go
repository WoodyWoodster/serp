package shared

import (
	"strconv"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type AppSyncEvent struct {
	FieldName  string                 `json:"fieldName"`
	Arguments  map[string]interface{} `json:"arguments"`
	Identity   map[string]interface{} `json:"identity"`
	Source     map[string]interface{} `json:"source"`
	Request    map[string]interface{} `json:"request"`
	PrevResult map[string]interface{} `json:"prevResult"`
}

type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "PENDING"
	OrderStatusConfirmed  OrderStatus = "CONFIRMED"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusShipped    OrderStatus = "SHIPPED"
	OrderStatusDelivered  OrderStatus = "DELIVERED"
	OrderStatusCancelled  OrderStatus = "CANCELLED"
)

type Order struct {
	ID          string      `json:"id"`
	CustomerID  string      `json:"customerId"`
	Status      OrderStatus `json:"status"`
	Items       []OrderItem `json:"items"`
	TotalAmount float64     `json:"totalAmount"`
	CreatedAt   string      `json:"createdAt"`
	UpdatedAt   string      `json:"updatedAt"`
}

type OrderItem struct {
	ID        string  `json:"id"`
	OrderID   string  `json:"orderId"`
	ItemID    string  `json:"itemId"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unitPrice"`
}

type OrderFilterInput struct {
	CustomerID string      `json:"customerId,omitempty"`
	Status     OrderStatus `json:"status,omitempty"`
}

type CreateOrderInput struct {
	CustomerID string                 `json:"customerId"`
	Items      []CreateOrderItemInput `json:"items"`
}

type CreateOrderItemInput struct {
	ItemID   string `json:"itemId"`
	Quantity int    `json:"quantity"`
}

type UpdateOrderStatusInput struct {
	ID     string      `json:"id"`
	Status OrderStatus `json:"status"`
}

func MarshalOrderItems(items []OrderItem) []types.AttributeValue {
	result := make([]types.AttributeValue, len(items))
	for i, item := range items {
		result[i] = &types.AttributeValueMemberM{
			Value: map[string]types.AttributeValue{
				"id":        &types.AttributeValueMemberS{Value: item.ID},
				"orderId":   &types.AttributeValueMemberS{Value: item.OrderID},
				"itemId":    &types.AttributeValueMemberS{Value: item.ItemID},
				"quantity":  &types.AttributeValueMemberN{Value: strconv.Itoa(item.Quantity)},
				"unitPrice": &types.AttributeValueMemberN{Value: strconv.FormatFloat(item.UnitPrice, 'f', 2, 64)},
			},
		}
	}
	return result
}
