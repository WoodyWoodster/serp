package shared

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsappsync"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsevents"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type SharedStackProps struct {
	awscdk.StackProps
}

func NewSharedStack(scope constructs.Construct, id string, props *SharedStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	vpc := awsec2.NewVpc(stack, jsii.String("ErpVpc"), &awsec2.VpcProps{
		MaxAzs:      jsii.Number(2),
		NatGateways: jsii.Number(1),
	})

	lambdaSecurityGroup := awsec2.NewSecurityGroup(stack, jsii.String("LambdaSecurityGroup"), &awsec2.SecurityGroupProps{
		Vpc:              vpc,
		Description:      jsii.String("Security group for Lambda functions"),
		AllowAllOutbound: jsii.Bool(true),
	})

	lambdaRole := awsiam.NewRole(stack, jsii.String("LambdaExecutionRole"), &awsiam.RoleProps{
		AssumedBy: awsiam.NewServicePrincipal(jsii.String("lambda.amazonaws.com"), nil),
		ManagedPolicies: &[]awsiam.IManagedPolicy{
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("service-role/AWSLambdaBasicExecutionRole")),
		},
	})

	eventBus := awsevents.NewEventBus(stack, jsii.String("ErpEventBus"), &awsevents.EventBusProps{
		EventBusName: jsii.String("erp-event-bus"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("ErpEventBusName"), &awscdk.CfnOutputProps{
		Value:       eventBus.EventBusName(),
		Description: jsii.String("The name of the ERP EventBus"),
		ExportName:  jsii.String("ErpEventBusName"),
	})

	api := awsappsync.NewGraphqlApi(stack, jsii.String("ErpApi"), &awsappsync.GraphqlApiProps{
		Name:   jsii.String("ErpApi"),
		Schema: awsappsync.SchemaFile_FromAsset(jsii.String("schema.graphql")),
		AuthorizationConfig: &awsappsync.AuthorizationConfig{
			DefaultAuthorization: &awsappsync.AuthorizationMode{
				AuthorizationType: awsappsync.AuthorizationType_API_KEY,
			},
		},
	})

	awscdk.NewCfnOutput(stack, jsii.String("VpcId"), &awscdk.CfnOutputProps{
		Value:       vpc.VpcId(),
		ExportName:  jsii.String("ErpVpcId"),
		Description: jsii.String("VPC ID for ERP services"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("LambdaSecurityGroupId"), &awscdk.CfnOutputProps{
		Value:       lambdaSecurityGroup.SecurityGroupId(),
		ExportName:  jsii.String("ErpLambdaSecurityGroupId"),
		Description: jsii.String("Security Group ID for Lambda functions"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("LambdaRoleArn"), &awscdk.CfnOutputProps{
		Value:       lambdaRole.RoleArn(),
		ExportName:  jsii.String("ErpLambdaRoleArn"),
		Description: jsii.String("IAM Role ARN for Lambda functions"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("GraphqlApiId"), &awscdk.CfnOutputProps{
		Value:       api.ApiId(),
		ExportName:  jsii.String("ErpGraphqlApiId"),
		Description: jsii.String("AppSync API ID for ERP services"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("GraphqlApiUrl"), &awscdk.CfnOutputProps{
		Value:       api.GraphqlUrl(),
		ExportName:  jsii.String("ErpGraphqlApiUrl"),
		Description: jsii.String("AppSync API URL for ERP services"),
	})

	return stack
}
