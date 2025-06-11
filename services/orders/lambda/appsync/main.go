package appsync

import (
	"context"
	"fmt"
	"time"

	"serp/services/orders/lambda/shared"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/google/uuid"
)

type Handler struct {
	db *shared.DB
	eb *eventbridge.Client
}

func NewHandler(ctx context.Context) (*Handler, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}
	return &Handler{
		db: shared.NewDB(cfg),
		eb: eventbridge.NewFromConfig(cfg),
	}, nil
}

func (h *Handler) HandleRequest(ctx context.Context, event shared.AppSyncEvent) (interface{}, error) {
	switch event.FieldName {
	case "getOrder":
		id, _ := event.Arguments["id"].(string)
		return h.getOrderByID(ctx, id)
	case "listOrders":
		return h.listOrders(ctx, event.Arguments)
	case "createOrder":
		return h.createOrder(ctx, event.Arguments)
	case "updateOrderStatus":
		return h.updateOrderStatus(ctx, event.Arguments)
	case "cancelOrder":
		id, _ := event.Arguments["id"].(string)
		return h.cancelOrder(ctx, id)
	default:
		return nil, fmt.Errorf("unknown field: %s", event.FieldName)
	}
}

func (h *Handler) getOrderByID(ctx context.Context, id string) (*shared.Order, error) {
	return h.db.GetOrder(ctx, id)
}

func (h *Handler) listOrders(ctx context.Context, args map[string]interface{}) ([]shared.Order, error) {
	return h.db.ListOrders(ctx)
}

func (h *Handler) createOrder(ctx context.Context, args map[string]interface{}) (*shared.Order, error) {
	input := args["input"].(map[string]interface{})
	now := time.Now().UTC()

	order := shared.Order{
		ID:         uuid.New().String(),
		CustomerID: input["customerId"].(string),
		Status:     shared.OrderStatusPending,
		Items:      make([]shared.OrderItem, 0),
		CreatedAt:  now.Format(time.RFC3339),
		UpdatedAt:  now.Format(time.RFC3339),
	}

	items := input["items"].([]interface{})
	for _, item := range items {
		itemMap := item.(map[string]interface{})
		orderItem := shared.OrderItem{
			ID:       uuid.New().String(),
			OrderID:  order.ID,
			ItemID:   itemMap["itemId"].(string),
			Quantity: int(itemMap["quantity"].(float64)),
		}
		order.Items = append(order.Items, orderItem)
	}

	return h.db.CreateOrder(ctx, order)
}

func (h *Handler) updateOrderStatus(ctx context.Context, args map[string]interface{}) (*shared.Order, error) {
	input := args["input"].(map[string]interface{})
	orderID := input["orderId"].(string)
	status := shared.OrderStatus(input["status"].(string))

	return h.db.UpdateOrderStatus(ctx, orderID, status)
}

func (h *Handler) cancelOrder(ctx context.Context, id string) (*shared.Order, error) {
	return h.db.UpdateOrderStatus(ctx, id, shared.OrderStatusCancelled)
}

func main() {
	handler, err := NewHandler(context.Background())
	if err != nil {
		panic(err)
	}

	lambda.Start(handler.HandleRequest)
}
