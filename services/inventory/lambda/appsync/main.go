package appsync

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"serp/services/inventory/lambda/shared"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/google/uuid"
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

func (h *Handler) HandleRequest(ctx context.Context, event shared.AppSyncEvent) (interface{}, error) {
	switch event.FieldName {
	case "getItem":
		id, _ := event.Arguments["id"].(string)
		return h.getItemByID(ctx, id)
	case "listItems":
		return h.listItems(ctx, event.Arguments)
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
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		return nil, fmt.Errorf("TABLE_NAME environment variable is not set")
	}

	result, err := h.ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %v", err)
	}
	if result.Item == nil {
		return nil, nil
	}

	item := shared.UnmarshalItem(result.Item)
	return &item, nil
}

func (h *Handler) listItems(ctx context.Context, args map[string]interface{}) ([]shared.Item, error) {
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		return nil, fmt.Errorf("TABLE_NAME environment variable is not set")
	}

	result, err := h.ddb.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan items: %v", err)
	}

	items := make([]shared.Item, 0, len(result.Items))
	for _, item := range result.Items {
		invItem := shared.UnmarshalItem(item)
		items = append(items, invItem)
	}

	return items, nil
}

func (h *Handler) createItem(ctx context.Context, args map[string]interface{}) (*shared.Item, error) {
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		return nil, fmt.Errorf("TABLE_NAME environment variable is not set")
	}

	now := time.Now().UTC()
	item := shared.Item{
		ID:          uuid.New().String(),
		Name:        args["name"].(string),
		Description: args["description"].(string),
		Quantity:    int(args["quantity"].(float64)),
		CreatedAt:   now.Format(time.RFC3339),
		UpdatedAt:   now.Format(time.RFC3339),
	}

	_, err := h.ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]types.AttributeValue{
			"id":          &types.AttributeValueMemberS{Value: item.ID},
			"name":        &types.AttributeValueMemberS{Value: item.Name},
			"description": &types.AttributeValueMemberS{Value: item.Description},
			"quantity":    &types.AttributeValueMemberN{Value: strconv.Itoa(item.Quantity)},
			"created_at":  &types.AttributeValueMemberS{Value: item.CreatedAt},
			"updated_at":  &types.AttributeValueMemberS{Value: item.UpdatedAt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create item: %v", err)
	}

	return &item, nil
}

func (h *Handler) updateItem(ctx context.Context, args map[string]interface{}) (*shared.Item, error) {
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		return nil, fmt.Errorf("TABLE_NAME environment variable is not set")
	}

	id := args["id"].(string)
	now := time.Now().UTC()

	updateExpr := "SET #name = :name, #description = :description, #quantity = :quantity, #updated_at = :updated_at"
	exprNames := map[string]string{
		"#name":        "name",
		"#description": "description",
		"#quantity":    "quantity",
		"#updated_at":  "updated_at",
	}
	exprValues := map[string]types.AttributeValue{
		":name":        &types.AttributeValueMemberS{Value: args["name"].(string)},
		":description": &types.AttributeValueMemberS{Value: args["description"].(string)},
		":quantity":    &types.AttributeValueMemberN{Value: strconv.Itoa(int(args["quantity"].(float64)))},
		":updated_at":  &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
	}

	result, err := h.ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(tableName),
		Key:                       map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: id}},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  exprNames,
		ExpressionAttributeValues: exprValues,
		ReturnValues:              types.ReturnValueAllNew,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update item: %v", err)
	}

	item := shared.UnmarshalItem(result.Attributes)
	return &item, nil
}

func (h *Handler) deleteItem(ctx context.Context, id string) (*shared.Item, error) {
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		return nil, fmt.Errorf("TABLE_NAME environment variable is not set")
	}

	item, err := h.getItemByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}

	_, err = h.ddb.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to delete item: %v", err)
	}

	return item, nil
}

func main() {
	handler, err := NewHandler(context.Background())
	if err != nil {
		panic(err)
	}

	lambda.Start(handler.HandleRequest)
}
