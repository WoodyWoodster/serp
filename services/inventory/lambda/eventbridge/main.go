package eventbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"serp/services/inventory/lambda/shared"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
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

func (h *Handler) HandleRequest(ctx context.Context, event events.CloudWatchEvent) error {
	var orderEvent shared.OrderEvent
	if err := json.Unmarshal([]byte(event.Detail), &orderEvent); err != nil {
		return fmt.Errorf("invalid event: %v", err)
	}

	switch event.DetailType {
	case "ORDER_CREATED":
		return h.handleOrderCreated(ctx, orderEvent)
	case "ORDER_CANCELLED":
		return h.handleOrderCancelled(ctx, orderEvent)
	default:
		return fmt.Errorf("unknown event type: %s", event.DetailType)
	}
}

func (h *Handler) handleOrderCreated(ctx context.Context, event shared.OrderEvent) error {
	item, err := h.db.GetItem(ctx, event.ItemID)
	if err != nil {
		return fmt.Errorf("failed to get item: %v", err)
	}
	if item.Quantity < event.Quantity {
		return h.sendInventoryEvent(ctx, "INSUFFICIENT_INVENTORY", event.OrderID, event.ItemID, event.Quantity)
	}
	item.Quantity -= event.Quantity
	_, err = h.db.UpdateItem(ctx, *item)
	if err != nil {
		return fmt.Errorf("failed to update inventory: %v", err)
	}
	return h.sendInventoryEvent(ctx, "INVENTORY_UPDATED", event.OrderID, event.ItemID, event.Quantity)
}

func (h *Handler) handleOrderCancelled(ctx context.Context, event shared.OrderEvent) error {
	item, err := h.db.GetItem(ctx, event.ItemID)
	if err != nil {
		return fmt.Errorf("failed to get item: %v", err)
	}
	item.Quantity += event.Quantity
	_, err = h.db.UpdateItem(ctx, *item)
	if err != nil {
		return fmt.Errorf("failed to restore inventory: %v", err)
	}
	return h.sendInventoryEvent(ctx, "INVENTORY_RESTORED", event.OrderID, event.ItemID, event.Quantity)
}

func (h *Handler) sendInventoryEvent(ctx context.Context, eventType, orderID, itemID string, quantity int) error {
	event := shared.OrderEvent{
		Type:      eventType,
		OrderID:   orderID,
		ItemID:    itemID,
		Quantity:  quantity,
		Timestamp: time.Now(),
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %v", err)
	}

	_, err = h.eb.PutEvents(ctx, &eventbridge.PutEventsInput{
		Entries: []eventbridgetypes.PutEventsRequestEntry{
			{
				Source:       aws.String("inventory.service"),
				DetailType:   aws.String(eventType),
				Detail:       aws.String(string(eventBytes)),
				EventBusName: aws.String(os.Getenv("EVENT_BUS_NAME")),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to send event: %v", err)
	}
	return nil
}

func main() {
	handler, err := NewHandler(context.Background())
	if err != nil {
		panic(err)
	}

	lambda.Start(handler.HandleRequest)
}
