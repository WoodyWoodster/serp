package main

import (
	"os"
	"serp/infrastructure/services"
	"serp/infrastructure/shared"

	"github.com/aws/aws-cdk-go/awscdk/v2"

	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type SerpStackProps struct {
	awscdk.StackProps
}

func NewSerpStack(scope constructs.Construct, id string, props *SerpStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	return stack
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	// Create shared infrastructure stack
	shared.NewSharedStack(app, "ErpSharedStack", &shared.SharedStackProps{
		StackProps: awscdk.StackProps{
			Env: env(),
		},
	})

	// Create Inventory microservice stack
	services.NewMicroserviceStack(app, "InventoryServiceStack", &services.MicroserviceStackProps{
		StackProps: awscdk.StackProps{
			Env: env(),
		},
		ServiceName:     "inventory",
		VpcId:           awscdk.Fn_ImportValue(jsii.String("ErpVpcId")),
		SecurityGroupId: awscdk.Fn_ImportValue(jsii.String("ErpLambdaSecurityGroupId")),
		LambdaRoleArn:   awscdk.Fn_ImportValue(jsii.String("ErpLambdaRoleArn")),
		GraphqlApiId:    awscdk.Fn_ImportValue(jsii.String("ErpGraphqlApiId")),
	})

	// Create Orders microservice stack
	services.NewMicroserviceStack(app, "OrdersServiceStack", &services.MicroserviceStackProps{
		StackProps: awscdk.StackProps{
			Env: env(),
		},
		ServiceName:     "orders",
		VpcId:           awscdk.Fn_ImportValue(jsii.String("ErpVpcId")),
		SecurityGroupId: awscdk.Fn_ImportValue(jsii.String("ErpLambdaSecurityGroupId")),
		LambdaRoleArn:   awscdk.Fn_ImportValue(jsii.String("ErpLambdaRoleArn")),
		GraphqlApiId:    awscdk.Fn_ImportValue(jsii.String("ErpGraphqlApiId")),
	})

	app.Synth(nil)
}

func env() *awscdk.Environment {
	return &awscdk.Environment{
		Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
		Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	}
}
