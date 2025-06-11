package appsync

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"serp/services/orders/lambda/shared"

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
		return nil, fmt.Errorf("failed to get order: %v", err)
	}
	if result.Item == nil {
		return nil, nil
	}

	order := shared.UnmarshalOrder(result.Item)
	return &order, nil
}

func (h *Handler) listOrders(ctx context.Context, args map[string]interface{}) ([]shared.Order, error) {
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		return nil, fmt.Errorf("TABLE_NAME environment variable is not set")
	}

	result, err := h.ddb.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan orders: %v", err)
	}

	orders := make([]shared.Order, 0, len(result.Items))
	for _, item := range result.Items {
		order := shared.UnmarshalOrder(item)
		orders = append(orders, order)
	}

	return orders, nil
}

func (h *Handler) createOrder(ctx context.Context, args map[string]interface{}) (*shared.Order, error) {
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		return nil, fmt.Errorf("TABLE_NAME environment variable is not set")
	}

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

	var totalAmount float64
	for _, item := range order.Items {
		totalAmount += float64(item.Quantity) * item.UnitPrice
	}
	order.TotalAmount = totalAmount

	_, err := h.ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]types.AttributeValue{
			"id":           &types.AttributeValueMemberS{Value: order.ID},
			"customer_id":  &types.AttributeValueMemberS{Value: order.CustomerID},
			"status":       &types.AttributeValueMemberS{Value: string(order.Status)},
			"items":        &types.AttributeValueMemberL{Value: shared.MarshalOrderItems(order.Items)},
			"total_amount": &types.AttributeValueMemberN{Value: strconv.FormatFloat(order.TotalAmount, 'f', 2, 64)},
			"created_at":   &types.AttributeValueMemberS{Value: order.CreatedAt},
			"updated_at":   &types.AttributeValueMemberS{Value: order.UpdatedAt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %v", err)
	}

	return &order, nil
}

func (h *Handler) updateOrderStatus(ctx context.Context, args map[string]interface{}) (*shared.Order, error) {
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		return nil, fmt.Errorf("TABLE_NAME environment variable is not set")
	}

	input := args["input"].(map[string]interface{})
	orderID := input["orderId"].(string)
	status := shared.OrderStatus(input["status"].(string))
	now := time.Now().UTC()

	updateExpr := "SET #status = :status, #updated_at = :updated_at"
	exprNames := map[string]string{
		"#status":     "status",
		"#updated_at": "updated_at",
	}
	exprValues := map[string]types.AttributeValue{
		":status":     &types.AttributeValueMemberS{Value: string(status)},
		":updated_at": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
	}

	result, err := h.ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(tableName),
		Key:                       map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: orderID}},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  exprNames,
		ExpressionAttributeValues: exprValues,
		ReturnValues:              types.ReturnValueAllNew,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update order status: %v", err)
	}

	order := shared.UnmarshalOrder(result.Attributes)
	return &order, nil
}

func (h *Handler) cancelOrder(ctx context.Context, id string) (*shared.Order, error) {
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		return nil, fmt.Errorf("TABLE_NAME environment variable is not set")
	}

	order, err := h.getOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, nil
	}

	now := time.Now().UTC()
	updateExpr := "SET #status = :status, #updated_at = :updated_at"
	exprNames := map[string]string{
		"#status":     "status",
		"#updated_at": "updated_at",
	}
	exprValues := map[string]types.AttributeValue{
		":status":     &types.AttributeValueMemberS{Value: string(shared.OrderStatusCancelled)},
		":updated_at": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
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
		return nil, fmt.Errorf("failed to cancel order: %v", err)
	}

	updatedOrder := shared.UnmarshalOrder(result.Attributes)
	return &updatedOrder, nil
}

func main() {
	handler, err := NewHandler(context.Background())
	if err != nil {
		panic(err)
	}

	lambda.Start(handler.HandleRequest)
}
