package shared

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

type DB struct {
	client *dynamodb.Client
}

func NewDB(cfg aws.Config) *DB {
	return &DB{
		client: dynamodb.NewFromConfig(cfg),
	}
}

func (db *DB) GetOrder(ctx context.Context, id string) (*Order, error) {
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		return nil, fmt.Errorf("TABLE_NAME environment variable is not set")
	}

	result, err := db.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", id)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", id)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %v", err)
	}
	if result.Item == nil {
		return nil, nil
	}

	order := UnmarshalOrder(result.Item)
	return &order, nil
}

func (db *DB) ListOrders(ctx context.Context) ([]Order, error) {
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		return nil, fmt.Errorf("TABLE_NAME environment variable is not set")
	}

	result, err := db.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(tableName),
		FilterExpression: aws.String("begins_with(PK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":prefix": &types.AttributeValueMemberS{Value: "ORDER#"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan orders: %v", err)
	}

	orders := make([]Order, 0, len(result.Items))
	for _, item := range result.Items {
		order := UnmarshalOrder(item)
		orders = append(orders, order)
	}

	return orders, nil
}

func (db *DB) CreateOrder(ctx context.Context, order Order) (*Order, error) {
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		return nil, fmt.Errorf("TABLE_NAME environment variable is not set")
	}

	if order.ID == "" {
		order.ID = uuid.New().String()
	}

	_, err := db.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]types.AttributeValue{
			"PK":           &types.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", order.ID)},
			"SK":           &types.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", order.ID)},
			"customer_id":  &types.AttributeValueMemberS{Value: order.CustomerID},
			"status":       &types.AttributeValueMemberS{Value: string(order.Status)},
			"total_amount": &types.AttributeValueMemberN{Value: strconv.FormatFloat(order.TotalAmount, 'f', 2, 64)},
			"created_at":   &types.AttributeValueMemberS{Value: order.CreatedAt},
			"updated_at":   &types.AttributeValueMemberS{Value: order.UpdatedAt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %v", err)
	}

	for _, item := range order.Items {
		_, err := db.client.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(tableName),
			Item: map[string]types.AttributeValue{
				"PK":         &types.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", order.ID)},
				"SK":         &types.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", item.ID)},
				"item_id":    &types.AttributeValueMemberS{Value: item.ID},
				"quantity":   &types.AttributeValueMemberN{Value: strconv.Itoa(item.Quantity)},
				"unit_price": &types.AttributeValueMemberN{Value: strconv.FormatFloat(item.UnitPrice, 'f', 2, 64)},
				"created_at": &types.AttributeValueMemberS{Value: order.CreatedAt},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create order item: %v", err)
		}
	}

	return &order, nil
}

func (db *DB) UpdateOrderStatus(ctx context.Context, id string, status OrderStatus) (*Order, error) {
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		return nil, fmt.Errorf("TABLE_NAME environment variable is not set")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	result, err := db.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", id)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", id)},
		},
		UpdateExpression: aws.String("SET #status = :status, #updated_at = :updated_at"),
		ExpressionAttributeNames: map[string]string{
			"#status":     "status",
			"#updated_at": "updated_at",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":     &types.AttributeValueMemberS{Value: string(status)},
			":updated_at": &types.AttributeValueMemberS{Value: now},
		},
		ReturnValues: types.ReturnValueAllNew,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update order status: %v", err)
	}

	order := UnmarshalOrder(result.Attributes)
	return &order, nil
}

func UnmarshalOrder(av map[string]types.AttributeValue) Order {
	order := Order{}
	if v, ok := av["ID"].(*types.AttributeValueMemberS); ok {
		order.ID = v.Value
	}
	if v, ok := av["customer_id"].(*types.AttributeValueMemberS); ok {
		order.CustomerID = v.Value
	}
	if v, ok := av["status"].(*types.AttributeValueMemberS); ok {
		order.Status = OrderStatus(v.Value)
	}
	if v, ok := av["total_amount"].(*types.AttributeValueMemberN); ok {
		if f, err := strconv.ParseFloat(v.Value, 64); err == nil {
			order.TotalAmount = f
		}
	}
	if v, ok := av["created_at"].(*types.AttributeValueMemberS); ok {
		order.CreatedAt = v.Value
	}
	if v, ok := av["updated_at"].(*types.AttributeValueMemberS); ok {
		order.UpdatedAt = v.Value
	}
	return order
}
