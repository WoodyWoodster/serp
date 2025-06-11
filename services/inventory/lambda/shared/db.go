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

func GetItemByID(ctx context.Context, db DBClient, id string) (Item, error) {
	result, err := db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &TableName,
		Key: map[string]dynamodbtypes.AttributeValue{
			"PK": &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", id)},
			"SK": &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", id)},
		},
	})
	if err != nil {
		return Item{}, err
	}
	if result.Item == nil {
		return Item{}, fmt.Errorf("Item not found")
	}
	return UnmarshalItem(result.Item), nil
}

func ListItems(ctx context.Context, db DBClient, filter ItemFilterInput) ([]Item, error) {
	queryInput := &dynamodb.QueryInput{
		TableName:              &TableName,
		KeyConditionExpression: aws.String("PK BEGINS_WITH :pk"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":pk": &dynamodbtypes.AttributeValueMemberS{Value: "ITEM#"},
		},
	}

	if filter.Category != "" {
		queryInput.FilterExpression = aws.String("category = :category")
		queryInput.ExpressionAttributeValues[":category"] = &dynamodbtypes.AttributeValueMemberS{Value: filter.Category}
	}

	result, err := db.Query(ctx, queryInput)
	if err != nil {
		return nil, err
	}

	items := make([]Item, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, UnmarshalItem(item))
	}

	return items, nil
}

func CreateItem(ctx context.Context, db DBClient, input CreateItemInput) (Item, error) {
	now := time.Now().UTC()
	id := uuid.New().String()
	item := Item{
		ID:          id,
		Name:        input.Name,
		Description: input.Description,
		Quantity:    input.Quantity,
		UnitPrice:   input.UnitPrice,
		Category:    input.Category,
		CreatedAt:   now.Format(time.RFC3339),
		UpdatedAt:   now.Format(time.RFC3339),
	}

	_, err := db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &TableName,
		Item: map[string]dynamodbtypes.AttributeValue{
			"PK":          &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", id)},
			"SK":          &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", id)},
			"Type":        &dynamodbtypes.AttributeValueMemberS{Value: "ITEM"},
			"ID":          &dynamodbtypes.AttributeValueMemberS{Value: item.ID},
			"Name":        &dynamodbtypes.AttributeValueMemberS{Value: item.Name},
			"Description": &dynamodbtypes.AttributeValueMemberS{Value: item.Description},
			"Quantity":    &dynamodbtypes.AttributeValueMemberN{Value: strconv.Itoa(item.Quantity)},
			"UnitPrice":   &dynamodbtypes.AttributeValueMemberN{Value: strconv.FormatFloat(item.UnitPrice, 'f', 2, 64)},
			"Category":    &dynamodbtypes.AttributeValueMemberS{Value: item.Category},
			"CreatedAt":   &dynamodbtypes.AttributeValueMemberS{Value: item.CreatedAt},
			"UpdatedAt":   &dynamodbtypes.AttributeValueMemberS{Value: item.UpdatedAt},
		},
	})
	if err != nil {
		return Item{}, err
	}

	_, err = db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &TableName,
		Item: map[string]dynamodbtypes.AttributeValue{
			"PK":        &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("METADATA#CATEGORY#%s", item.Category)},
			"SK":        &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", id)},
			"Type":      &dynamodbtypes.AttributeValueMemberS{Value: "METADATA"},
			"ItemID":    &dynamodbtypes.AttributeValueMemberS{Value: id},
			"CreatedAt": &dynamodbtypes.AttributeValueMemberS{Value: item.CreatedAt},
		},
	})
	if err != nil {
		return Item{}, err
	}

	return item, nil
}

func UpdateItem(ctx context.Context, db DBClient, input UpdateItemInput) (Item, error) {
	now := time.Now().UTC()
	updateExpr := "SET #name = :name, #description = :description, #quantity = :quantity, #unitPrice = :unitPrice, #category = :category, #updated_at = :updated_at"
	exprNames := map[string]string{
		"#name":        "Name",
		"#description": "Description",
		"#quantity":    "Quantity",
		"#unitPrice":   "UnitPrice",
		"#category":    "Category",
		"#updated_at":  "UpdatedAt",
	}
	exprValues := map[string]dynamodbtypes.AttributeValue{
		":name":        &dynamodbtypes.AttributeValueMemberS{Value: input.Name},
		":description": &dynamodbtypes.AttributeValueMemberS{Value: input.Description},
		":quantity":    &dynamodbtypes.AttributeValueMemberN{Value: strconv.Itoa(input.Quantity)},
		":unitPrice":   &dynamodbtypes.AttributeValueMemberN{Value: strconv.FormatFloat(input.UnitPrice, 'f', 2, 64)},
		":category":    &dynamodbtypes.AttributeValueMemberS{Value: input.Category},
		":updated_at":  &dynamodbtypes.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
	}

	result, err := db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 &TableName,
		Key:                       map[string]dynamodbtypes.AttributeValue{"PK": &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", input.ID)}, "SK": &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", input.ID)}},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  exprNames,
		ExpressionAttributeValues: exprValues,
		ReturnValues:              dynamodbtypes.ReturnValueAllNew,
	})
	if err != nil {
		return Item{}, err
	}

	item := UnmarshalItem(result.Attributes)

	if input.Category != "" {
		_, err = db.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: &TableName,
			Item: map[string]dynamodbtypes.AttributeValue{
				"PK":        &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("METADATA#CATEGORY#%s", input.Category)},
				"SK":        &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("ITEM#%s", input.ID)},
				"Type":      &dynamodbtypes.AttributeValueMemberS{Value: "METADATA"},
				"ItemID":    &dynamodbtypes.AttributeValueMemberS{Value: input.ID},
				"CreatedAt": &dynamodbtypes.AttributeValueMemberS{Value: item.CreatedAt},
			},
		})
		if err != nil {
			return Item{}, err
		}
	}

	return item, nil
}

func UnmarshalItem(av map[string]dynamodbtypes.AttributeValue) Item {
	item := Item{}
	if v, ok := av["ID"].(*dynamodbtypes.AttributeValueMemberS); ok {
		item.ID = v.Value
	}
	if v, ok := av["Name"].(*dynamodbtypes.AttributeValueMemberS); ok {
		item.Name = v.Value
	}
	if v, ok := av["Description"].(*dynamodbtypes.AttributeValueMemberS); ok {
		item.Description = v.Value
	}
	if v, ok := av["Quantity"].(*dynamodbtypes.AttributeValueMemberN); ok {
		if i, err := strconv.Atoi(v.Value); err == nil {
			item.Quantity = i
		}
	}
	if v, ok := av["UnitPrice"].(*dynamodbtypes.AttributeValueMemberN); ok {
		if f, err := strconv.ParseFloat(v.Value, 64); err == nil {
			item.UnitPrice = f
		}
	}
	if v, ok := av["Category"].(*dynamodbtypes.AttributeValueMemberS); ok {
		item.Category = v.Value
	}
	if v, ok := av["CreatedAt"].(*dynamodbtypes.AttributeValueMemberS); ok {
		item.CreatedAt = v.Value
	}
	if v, ok := av["UpdatedAt"].(*dynamodbtypes.AttributeValueMemberS); ok {
		item.UpdatedAt = v.Value
	}
	return item
}
