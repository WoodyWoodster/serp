package eventbridge

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
)

type Handler struct {
	ddb *dynamodb.Client
	eb  *eventbridge.Client
}

func NewHandler(ctx context.Context) (*Handler, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}
	return &Handler{
		ddb: dynamodb.NewFromConfig(cfg),
		eb:  eventbridge.NewFromConfig(cfg),
	}, nil
}

func (h *Handler) HandleRequest(ctx context.Context, event events.CloudWatchEvent) error {
	switch event.DetailType {
	case "InventoryUpdated":
		return h.handleInventoryUpdated(ctx, event)
	default:
		return fmt.Errorf("unknown event type: %s", event.DetailType)
	}
}

func (h *Handler) handleInventoryUpdated(ctx context.Context, event events.CloudWatchEvent) error {
	var detail struct {
		ItemID   string `json:"itemId"`
		Quantity int    `json:"quantity"`
	}
	if err := json.Unmarshal([]byte(event.Detail), &detail); err != nil {
		return fmt.Errorf("failed to unmarshal event detail: %v", err)
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
