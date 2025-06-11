# Serverless ERP Application

This is a serverless ERP application built using AWS CDK. The application follows a microservices architecture where each ERP module is implemented as a separate microservice, all connected through a unified GraphQL API.

## Architecture

The application uses the following AWS services:
- AWS Lambda for serverless compute
- Amazon DynamoDB for data storage (using single-table design)
- Amazon AppSync for unified GraphQL API
- Amazon EventBridge for event processing
- Amazon Redshift for data warehousing
- Amazon OpenSearch for search functionality

## Project Structure

```
.
├── infrastructure/
│   ├── shared/           # Shared infrastructure components
│   │   └── shared_stack.go
│   └── services/         # Individual microservice stacks
│       └── microservice_stack.go
├── services/            # Microservice implementations
│   ├── inventory/       # Inventory service
│   │   └── lambda/      # Lambda function code
│   └── orders/          # Orders service
│       └── lambda/      # Lambda function code
├── schema.graphql      # Unified GraphQL schema
├── serp.go            # Main CDK application
└── README.md
```

## Adding a New Microservice

To add a new microservice:

1. Create a new directory under `services/` for your microservice (e.g., `services/orders/`)
2. Implement your Lambda function code in the `lambda/` directory
3. Add your types and operations to the unified `schema.graphql` file
4. Add a new stack instance in `serp.go`:

```go
services.NewMicroserviceStack(app, "YourServiceStack", &services.MicroserviceStackProps{
    StackProps: awscdk.StackProps{
        Env: env(),
    },
    ServiceName:     "YourService",
    VpcId:           awscdk.Fn_ImportValue(jsii.String("ErpVpcId")),
    SecurityGroupId: awscdk.Fn_ImportValue(jsii.String("ErpLambdaSecurityGroupId")),
    LambdaRoleArn:   awscdk.Fn_ImportValue(jsii.String("ErpLambdaRoleArn")),
    GraphqlApiId:    awscdk.Fn_ImportValue(jsii.String("ErpGraphqlApiId")),
})
```

## Development

1. Install dependencies:
```bash
go mod tidy
```

2. Deploy the stack:
```bash
cdk deploy
```

## GraphQL API

The application exposes a single GraphQL endpoint that combines all microservices. The endpoint URL will be available as a CloudFormation output after deployment.

Example queries:
```graphql
# Query inventory items
query {
  listItems {
    items {
      id
      name
      quantity
      unitPrice
    }
  }
}

# Create an order
mutation {
  createOrder(input: {
    customerId: "123"
    items: [
      {
        itemId: "456"
        quantity: 2
      }
    ]
  }) {
    id
    status
    totalAmount
  }
}
```
