package appsync

import (
	"context"
	"fmt"
	"time"

	"serp/services/inventory/lambda/shared"

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
	case "getItem":
		id, _ := event.Arguments["id"].(string)
		return h.getItemByID(ctx, id)
	case "listItems":
		return h.listItems(ctx)
	case "createItem":
		return h.createItem(ctx, event.Arguments)
	case "updateItem":
		return h.updateItem(ctx, event.Arguments)
	case "deleteItem":
		id, _ := event.Arguments["id"].(string)
		return h.deleteItem(ctx, id)
	default:
		return nil, fmt.Errorf("unknown field: %s", event.FieldName)
	}
}

func (h *Handler) getItemByID(ctx context.Context, id string) (*shared.Item, error) {
	return h.db.GetItem(ctx, id)
}

func (h *Handler) listItems(ctx context.Context) ([]shared.Item, error) {
	return h.db.ListItems(ctx)
}

func (h *Handler) createItem(ctx context.Context, args map[string]any) (*shared.Item, error) {
	now := time.Now().UTC()
	item := shared.Item{
		ID:          uuid.New().String(),
		Name:        args["name"].(string),
		Description: args["description"].(string),
		Quantity:    int(args["quantity"].(float64)),
		CreatedAt:   now.Format(time.RFC3339),
		UpdatedAt:   now.Format(time.RFC3339),
	}

	return h.db.CreateItem(ctx, item)
}

func (h *Handler) updateItem(ctx context.Context, args map[string]interface{}) (*shared.Item, error) {
	now := time.Now().UTC()
	item := shared.Item{
		ID:          args["id"].(string),
		Name:        args["name"].(string),
		Description: args["description"].(string),
		Quantity:    int(args["quantity"].(float64)),
		UpdatedAt:   now.Format(time.RFC3339),
	}

	return h.db.UpdateItem(ctx, item)
}

func (h *Handler) deleteItem(ctx context.Context, id string) (*shared.Item, error) {
	return h.db.DeleteItem(ctx, id)
}

func main() {
	handler, err := NewHandler(context.Background())
	if err != nil {
		panic(err)
	}

	lambda.Start(handler.HandleRequest)
}
