package services

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsappsync"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsevents"
	"github.com/aws/aws-cdk-go/awscdk/v2/awseventstargets"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type MicroserviceStackProps struct {
	awscdk.StackProps
	ServiceName     string
	VpcId           *string
	SecurityGroupId *string
	LambdaRoleArn   *string
	GraphqlApiId    *string
}

func NewMicroserviceStack(scope constructs.Construct, id string, props *MicroserviceStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	table := awsdynamodb.NewTable(stack, jsii.String(props.ServiceName+"Table"), &awsdynamodb.TableProps{
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		BillingMode: awsdynamodb.BillingMode_PAY_PER_REQUEST,
		Stream:      awsdynamodb.StreamViewType_NEW_AND_OLD_IMAGES,
	})

	api := awsappsync.GraphqlApi_FromGraphqlApiAttributes(stack, jsii.String(props.ServiceName+"Api"), &awsappsync.GraphqlApiAttributes{
		GraphqlApiId:  props.GraphqlApiId,
		GraphqlApiArn: awscdk.Fn_ImportValue(jsii.String("ErpGraphqlApiUrl")),
	})

	function := awslambda.NewFunction(stack, jsii.String(props.ServiceName+"Function"), &awslambda.FunctionProps{
		Runtime: awslambda.Runtime_PROVIDED_AL2023(),
		Handler: jsii.String("main"),
		Code:    awslambda.Code_FromAsset(jsii.String("services/"+props.ServiceName+"/lambda"), nil),
		Environment: &map[string]*string{
			"TABLE_NAME": table.TableName(),
			"API_URL":    awscdk.Fn_ImportValue(jsii.String("ErpGraphqlApiUrl")),
		},
	})

	table.GrantReadWriteData(function)
	api.GrantMutation(function, jsii.String("*"))

	rule := awsevents.NewRule(stack, jsii.String(props.ServiceName+"StreamRule"), &awsevents.RuleProps{
		EventPattern: &awsevents.EventPattern{
			Source:     jsii.Strings("aws.dynamodb"),
			DetailType: jsii.Strings("AWS API Call via CloudTrail"),
			Detail: &map[string]interface{}{
				"eventSource": []string{"dynamodb.amazonaws.com"},
				"eventName":   []string{"PutItem", "UpdateItem", "DeleteItem"},
				"requestParameters": map[string]interface{}{
					"tableName": []string{*table.TableName()},
				},
			},
		},
	})

	rule.AddTarget(awseventstargets.NewLambdaFunction(function, nil))

	lambdaDataSource := api.AddLambdaDataSource(jsii.String(props.ServiceName+"LambdaDataSource"), function, nil)

	schemaPath := "services/" + props.ServiceName + "/schema.graphql"
	schemaBytes, err := os.ReadFile(schemaPath)
	if err == nil {
		schema := string(schemaBytes)
		if strings.Contains(schema, "type Query") {
			for _, field := range []string{"getItem", "listItems", "getOrder", "listOrders"} {
				lambdaDataSource.CreateResolver(
					jsii.String(props.ServiceName+field+"Resolver"),
					&awsappsync.BaseResolverProps{
						TypeName:  jsii.String("Query"),
						FieldName: jsii.String(field),
					},
				)
			}
		}
		if strings.Contains(schema, "type Mutation") {
			for _, field := range []string{"createItem", "updateItem", "deleteItem", "createOrder", "updateOrderStatus", "cancelOrder"} {
				lambdaDataSource.CreateResolver(
					jsii.String(props.ServiceName+field+"Resolver"),
					&awsappsync.BaseResolverProps{
						TypeName:  jsii.String("Mutation"),
						FieldName: jsii.String(field),
					},
				)
			}
		}
	}

	serviceResolvers := map[string]struct {
		Queries   []string
		Mutations []string
	}{
		"inventory": {
			Queries:   []string{"getItem", "listItems"},
			Mutations: []string{"createItem", "updateItem", "deleteItem"},
		},
		"orders": {
			Queries:   []string{"getOrder", "listOrders"},
			Mutations: []string{"createOrder", "updateOrderStatus", "cancelOrder"},
		},
	}

	resolvers, ok := serviceResolvers[props.ServiceName]
	if ok {
		for _, field := range resolvers.Queries {
			resolverId := fmt.Sprintf("%s_%s_%s_Resolver", props.ServiceName, "Query", field)
			lambdaDataSource.CreateResolver(
				jsii.String(resolverId),
				&awsappsync.BaseResolverProps{
					TypeName:  jsii.String("Query"),
					FieldName: jsii.String(field),
				},
			)
		}
		for _, field := range resolvers.Mutations {
			resolverId := fmt.Sprintf("%s_%s_%s_Resolver", props.ServiceName, "Mutation", field)
			lambdaDataSource.CreateResolver(
				jsii.String(resolverId),
				&awsappsync.BaseResolverProps{
					TypeName:  jsii.String("Mutation"),
					FieldName: jsii.String(field),
				},
			)
		}
	}

	return stack
}
