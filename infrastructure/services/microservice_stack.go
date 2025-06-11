package services

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsappsync"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambdaeventsources"
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

	environment := &map[string]*string{
		"TABLE_NAME":   table.TableName(),
		"API_URL":      awscdk.Fn_ImportValue(jsii.String("ErpGraphqlApiUrl")),
		"SERVICE_NAME": jsii.String(props.ServiceName),
		"EVENT_BUS":    awscdk.Fn_ImportValue(jsii.String("ErpEventBusName")),
		"LOG_LEVEL":    jsii.String("INFO"),
	}

	function := awslambda.NewFunction(stack, jsii.String(props.ServiceName+"Function"), &awslambda.FunctionProps{
		Runtime:     awslambda.Runtime_PROVIDED_AL2023(),
		Handler:     jsii.String("main"),
		Code:        awslambda.Code_FromAsset(jsii.String("services/"+props.ServiceName+"/lambda"), nil),
		Environment: environment,
		Timeout:     awscdk.Duration_Seconds(jsii.Number(30)),
		MemorySize:  jsii.Number(256),
	})

	table.GrantReadWriteData(function)
	api.GrantMutation(function, jsii.String("*"))

	streamEventSource := awslambdaeventsources.NewDynamoEventSource(table, &awslambdaeventsources.DynamoEventSourceProps{
		StartingPosition: awslambda.StartingPosition_LATEST,
		BatchSize:        jsii.Number(1),
		RetryAttempts:    jsii.Number(3),
	})

	function.AddEventSource(streamEventSource)

	lambdaDataSource := api.AddLambdaDataSource(jsii.String(props.ServiceName+"LambdaDataSource"), function, nil)

	switch props.ServiceName {
	case "inventory":
		lambdaDataSource.CreateResolver(
			jsii.String(props.ServiceName+"QueryResolver"),
			&awsappsync.BaseResolverProps{
				TypeName:  jsii.String("Query"),
				FieldName: jsii.String("inventory"),
			},
		)
		lambdaDataSource.CreateResolver(
			jsii.String(props.ServiceName+"InventoryQueriesResolver"),
			&awsappsync.BaseResolverProps{
				TypeName:  jsii.String("InventoryQueries"),
				FieldName: jsii.String("getItem"),
			},
		)
		lambdaDataSource.CreateResolver(
			jsii.String(props.ServiceName+"InventoryQueriesListResolver"),
			&awsappsync.BaseResolverProps{
				TypeName:  jsii.String("InventoryQueries"),
				FieldName: jsii.String("listItems"),
			},
		)
		lambdaDataSource.CreateResolver(
			jsii.String(props.ServiceName+"MutationResolver"),
			&awsappsync.BaseResolverProps{
				TypeName:  jsii.String("Mutation"),
				FieldName: jsii.String("inventory"),
			},
		)
		lambdaDataSource.CreateResolver(
			jsii.String(props.ServiceName+"InventoryMutationsResolver"),
			&awsappsync.BaseResolverProps{
				TypeName:  jsii.String("InventoryMutations"),
				FieldName: jsii.String("createItem"),
			},
		)
		lambdaDataSource.CreateResolver(
			jsii.String(props.ServiceName+"InventoryMutationsUpdateResolver"),
			&awsappsync.BaseResolverProps{
				TypeName:  jsii.String("InventoryMutations"),
				FieldName: jsii.String("updateItem"),
			},
		)
		lambdaDataSource.CreateResolver(
			jsii.String(props.ServiceName+"InventoryMutationsDeleteResolver"),
			&awsappsync.BaseResolverProps{
				TypeName:  jsii.String("InventoryMutations"),
				FieldName: jsii.String("deleteItem"),
			},
		)
	case "orders":
		lambdaDataSource.CreateResolver(
			jsii.String(props.ServiceName+"QueryResolver"),
			&awsappsync.BaseResolverProps{
				TypeName:  jsii.String("Query"),
				FieldName: jsii.String("orders"),
			},
		)
		lambdaDataSource.CreateResolver(
			jsii.String(props.ServiceName+"OrderQueriesResolver"),
			&awsappsync.BaseResolverProps{
				TypeName:  jsii.String("OrderQueries"),
				FieldName: jsii.String("getOrder"),
			},
		)
		lambdaDataSource.CreateResolver(
			jsii.String(props.ServiceName+"OrderQueriesListResolver"),
			&awsappsync.BaseResolverProps{
				TypeName:  jsii.String("OrderQueries"),
				FieldName: jsii.String("listOrders"),
			},
		)
		lambdaDataSource.CreateResolver(
			jsii.String(props.ServiceName+"MutationResolver"),
			&awsappsync.BaseResolverProps{
				TypeName:  jsii.String("Mutation"),
				FieldName: jsii.String("orders"),
			},
		)
		lambdaDataSource.CreateResolver(
			jsii.String(props.ServiceName+"OrderMutationsResolver"),
			&awsappsync.BaseResolverProps{
				TypeName:  jsii.String("OrderMutations"),
				FieldName: jsii.String("createOrder"),
			},
		)
		lambdaDataSource.CreateResolver(
			jsii.String(props.ServiceName+"OrderMutationsUpdateResolver"),
			&awsappsync.BaseResolverProps{
				TypeName:  jsii.String("OrderMutations"),
				FieldName: jsii.String("updateOrderStatus"),
			},
		)
		lambdaDataSource.CreateResolver(
			jsii.String(props.ServiceName+"OrderMutationsCancelResolver"),
			&awsappsync.BaseResolverProps{
				TypeName:  jsii.String("OrderMutations"),
				FieldName: jsii.String("cancelOrder"),
			},
		)
	}

	return stack
}
