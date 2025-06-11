package shared

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DB struct {
	client *dynamodb.Client
}

func NewDB(cfg aws.Config) *DB {
	return &DB{
		client: dynamodb.NewFromConfig(cfg),
	}
}

func (db *DB) GetItem(ctx context.Context, id string) (*Item, error) {
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		return nil, fmt.Errorf("TABLE_NAME environment variable is not set")
	}

	result, err := db.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", id)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", id)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %v", err)
	}
	if result.Item == nil {
		return nil, nil
	}

	item := UnmarshalItem(result.Item)
	return &item, nil
}

func (db *DB) ListItems(ctx context.Context) ([]Item, error) {
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		return nil, fmt.Errorf("TABLE_NAME environment variable is not set")
	}

	result, err := db.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(tableName),
		FilterExpression: aws.String("begins_with(PK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":prefix": &types.AttributeValueMemberS{Value: "ITEM#"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan items: %v", err)
	}

	items := make([]Item, 0, len(result.Items))
	for _, item := range result.Items {
		invItem := UnmarshalItem(item)
		items = append(items, invItem)
	}

	return items, nil
}

func (db *DB) CreateItem(ctx context.Context, item Item) (*Item, error) {
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		return nil, fmt.Errorf("TABLE_NAME environment variable is not set")
	}

	_, err := db.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]types.AttributeValue{
			"PK":          &types.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", item.ID)},
			"SK":          &types.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", item.ID)},
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

func (db *DB) UpdateItem(ctx context.Context, item Item) (*Item, error) {
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		return nil, fmt.Errorf("TABLE_NAME environment variable is not set")
	}

	updateExpr := "SET #name = :name, #description = :description, #quantity = :quantity, #updated_at = :updated_at"
	exprNames := map[string]string{
		"#name":        "name",
		"#description": "description",
		"#quantity":    "quantity",
		"#updated_at":  "updated_at",
	}
	exprValues := map[string]types.AttributeValue{
		":name":        &types.AttributeValueMemberS{Value: item.Name},
		":description": &types.AttributeValueMemberS{Value: item.Description},
		":quantity":    &types.AttributeValueMemberN{Value: strconv.Itoa(item.Quantity)},
		":updated_at":  &types.AttributeValueMemberS{Value: item.UpdatedAt},
	}

	result, err := db.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", item.ID)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", item.ID)},
		},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  exprNames,
		ExpressionAttributeValues: exprValues,
		ReturnValues:              types.ReturnValueAllNew,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update item: %v", err)
	}

	updatedItem := UnmarshalItem(result.Attributes)
	return &updatedItem, nil
}

func (db *DB) DeleteItem(ctx context.Context, id string) (*Item, error) {
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		return nil, fmt.Errorf("TABLE_NAME environment variable is not set")
	}

	item, err := db.GetItem(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}

	_, err = db.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", id)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", id)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to delete item: %v", err)
	}

	return item, nil
}

func UnmarshalItem(av map[string]types.AttributeValue) Item {
	item := Item{}
	if v, ok := av["ID"].(*types.AttributeValueMemberS); ok {
		item.ID = v.Value
	}
	if v, ok := av["Name"].(*types.AttributeValueMemberS); ok {
		item.Name = v.Value
	}
	if v, ok := av["Description"].(*types.AttributeValueMemberS); ok {
		item.Description = v.Value
	}
	if v, ok := av["Quantity"].(*types.AttributeValueMemberN); ok {
		if i, err := strconv.Atoi(v.Value); err == nil {
			item.Quantity = i
		}
	}
	if v, ok := av["UnitPrice"].(*types.AttributeValueMemberN); ok {
		if f, err := strconv.ParseFloat(v.Value, 64); err == nil {
			item.UnitPrice = f
		}
	}
	if v, ok := av["Category"].(*types.AttributeValueMemberS); ok {
		item.Category = v.Value
	}
	if v, ok := av["CreatedAt"].(*types.AttributeValueMemberS); ok {
		item.CreatedAt = v.Value
	}
	if v, ok := av["UpdatedAt"].(*types.AttributeValueMemberS); ok {
		item.UpdatedAt = v.Value
	}
	return item
}
