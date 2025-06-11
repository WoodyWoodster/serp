package shared

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

var TableName = os.Getenv("TABLE_NAME")

type DBClient interface {
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
}

func GetOrderByID(ctx context.Context, db DBClient, id string) (Order, error) {
	result, err := db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &TableName,
		Key: map[string]dynamodbtypes.AttributeValue{
			"PK": &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", id)},
			"SK": &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", id)},
		},
	})
	if err != nil {
		return Order{}, err
	}
	if result.Item == nil {
		return Order{}, fmt.Errorf("Order not found")
	}
	return UnmarshalOrder(result.Item), nil
}

func ListOrders(ctx context.Context, db DBClient, filter OrderFilterInput) ([]Order, error) {
	queryInput := &dynamodb.QueryInput{
		TableName:              &TableName,
		KeyConditionExpression: aws.String("PK BEGINS_WITH :pk"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":pk": &dynamodbtypes.AttributeValueMemberS{Value: "ORDER#"},
		},
	}

	if filter.CustomerID != "" {
		queryInput.FilterExpression = aws.String("customer_id = :customer_id")
		queryInput.ExpressionAttributeValues[":customer_id"] = &dynamodbtypes.AttributeValueMemberS{Value: filter.CustomerID}
	}
	if filter.Status != "" {
		queryInput.FilterExpression = aws.String("status = :status")
		queryInput.ExpressionAttributeValues[":status"] = &dynamodbtypes.AttributeValueMemberS{Value: string(filter.Status)}
	}

	result, err := db.Query(ctx, queryInput)
	if err != nil {
		return nil, err
	}

	orders := make([]Order, 0, len(result.Items))
	for _, item := range result.Items {
		orders = append(orders, UnmarshalOrder(item))
	}

	return orders, nil
}

func ListOrdersByCustomer(ctx context.Context, db DBClient, customerID string) ([]Order, error) {
	result, err := db.Query(ctx, &dynamodb.QueryInput{
		TableName:              &TableName,
		IndexName:              aws.String("CustomerIndex"),
		KeyConditionExpression: aws.String("CustomerID = :customerID"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":customerID": &dynamodbtypes.AttributeValueMemberS{Value: customerID},
		},
	})
	if err != nil {
		return nil, err
	}

	orders := make([]Order, 0, len(result.Items))
	for _, item := range result.Items {
		orders = append(orders, UnmarshalOrder(item))
	}
	return orders, nil
}

func CreateOrder(ctx context.Context, db DBClient, input CreateOrderInput) (Order, error) {
	now := time.Now().UTC()
	id := uuid.New().String()
	order := Order{
		ID:         id,
		CustomerID: input.CustomerID,
		Status:     "PENDING",
		Items:      make([]OrderItem, len(input.Items)),
		CreatedAt:  now.Format(time.RFC3339),
		UpdatedAt:  now.Format(time.RFC3339),
	}

	for i, item := range input.Items {
		order.Items[i] = OrderItem{
			ItemID:   item.ItemID,
			Quantity: item.Quantity,
		}
	}

	_, err := db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &TableName,
		Item: map[string]dynamodbtypes.AttributeValue{
			"PK":         &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", id)},
			"SK":         &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", id)},
			"Type":       &dynamodbtypes.AttributeValueMemberS{Value: "ORDER"},
			"ID":         &dynamodbtypes.AttributeValueMemberS{Value: order.ID},
			"CustomerID": &dynamodbtypes.AttributeValueMemberS{Value: order.CustomerID},
			"Status":     &dynamodbtypes.AttributeValueMemberS{Value: string(order.Status)},
			"Items":      &dynamodbtypes.AttributeValueMemberL{Value: MarshalOrderItems(order.Items)},
			"CreatedAt":  &dynamodbtypes.AttributeValueMemberS{Value: order.CreatedAt},
			"UpdatedAt":  &dynamodbtypes.AttributeValueMemberS{Value: order.UpdatedAt},
		},
	})
	if err != nil {
		return Order{}, err
	}

	_, err = db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &TableName,
		Item: map[string]dynamodbtypes.AttributeValue{
			"PK":        &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("CUSTOMER#%s", order.CustomerID)},
			"SK":        &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", id)},
			"Type":      &dynamodbtypes.AttributeValueMemberS{Value: "CUSTOMER_ORDER"},
			"OrderID":   &dynamodbtypes.AttributeValueMemberS{Value: id},
			"Status":    &dynamodbtypes.AttributeValueMemberS{Value: string(order.Status)},
			"CreatedAt": &dynamodbtypes.AttributeValueMemberS{Value: order.CreatedAt},
		},
	})
	if err != nil {
		return Order{}, err
	}

	return order, nil
}

func UpdateOrderStatus(ctx context.Context, db DBClient, input UpdateOrderStatusInput) (Order, error) {
	now := time.Now().UTC()
	updateExpr := "SET #status = :status, #updated_at = :updated_at"
	exprNames := map[string]string{
		"#status":     "Status",
		"#updated_at": "UpdatedAt",
	}
	exprValues := map[string]dynamodbtypes.AttributeValue{
		":status":     &dynamodbtypes.AttributeValueMemberS{Value: string(input.Status)},
		":updated_at": &dynamodbtypes.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
	}

	result, err := db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 &TableName,
		Key:                       map[string]dynamodbtypes.AttributeValue{"PK": &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", input.ID)}, "SK": &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", input.ID)}},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  exprNames,
		ExpressionAttributeValues: exprValues,
		ReturnValues:              dynamodbtypes.ReturnValueAllNew,
	})
	if err != nil {
		return Order{}, err
	}

	order := UnmarshalOrder(result.Attributes)

	_, err = db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &TableName,
		Item: map[string]dynamodbtypes.AttributeValue{
			"PK":        &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("CUSTOMER#%s", order.CustomerID)},
			"SK":        &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", input.ID)},
			"Type":      &dynamodbtypes.AttributeValueMemberS{Value: "CUSTOMER_ORDER"},
			"OrderID":   &dynamodbtypes.AttributeValueMemberS{Value: input.ID},
			"Status":    &dynamodbtypes.AttributeValueMemberS{Value: string(input.Status)},
			"CreatedAt": &dynamodbtypes.AttributeValueMemberS{Value: order.CreatedAt},
		},
	})
	if err != nil {
		return Order{}, err
	}

	return order, nil
}

func UnmarshalOrder(av map[string]dynamodbtypes.AttributeValue) Order {
	order := Order{}
	if v, ok := av["ID"].(*dynamodbtypes.AttributeValueMemberS); ok {
		order.ID = v.Value
	}
	if v, ok := av["CustomerID"].(*dynamodbtypes.AttributeValueMemberS); ok {
		order.CustomerID = v.Value
	}
	if v, ok := av["Status"].(*dynamodbtypes.AttributeValueMemberS); ok {
		order.Status = OrderStatus(v.Value)
	}
	if v, ok := av["TotalAmount"].(*dynamodbtypes.AttributeValueMemberN); ok {
		if f, err := strconv.ParseFloat(v.Value, 64); err == nil {
			order.TotalAmount = f
		}
	}
	if v, ok := av["CreatedAt"].(*dynamodbtypes.AttributeValueMemberS); ok {
		order.CreatedAt = v.Value
	}
	if v, ok := av["UpdatedAt"].(*dynamodbtypes.AttributeValueMemberS); ok {
		order.UpdatedAt = v.Value
	}
	return order
}
