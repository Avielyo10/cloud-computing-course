package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/apigateway"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/dynamodb"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lambda"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Set the AWS region explicitly
		awsRegion := pulumi.String("il-central-1")
		
		// DynamoDB table definition matching Go data model
		table, err := dynamodb.NewTable(ctx, "parkingTickets", &dynamodb.TableArgs{
			Attributes: dynamodb.TableAttributeArray{
				&dynamodb.TableAttributeArgs{Name: pulumi.String("ticketId"), Type: pulumi.String("S")},
				&dynamodb.TableAttributeArgs{Name: pulumi.String("plate"), Type: pulumi.String("S")},
				&dynamodb.TableAttributeArgs{Name: pulumi.String("parkingLot"), Type: pulumi.String("N")},
				&dynamodb.TableAttributeArgs{Name: pulumi.String("entryTime"), Type: pulumi.String("S")},
			},
			HashKey:     pulumi.String("ticketId"),
			BillingMode: pulumi.String("PAY_PER_REQUEST"),
		})
		if err != nil {
			return err
		}

		// IAM Role for Lambda functions
		lambdaRole, err := iam.NewRole(ctx, "lambdaRole", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Action": "sts:AssumeRole",
					"Principal": {"Service": "lambda.amazonaws.com"},
					"Effect": "Allow"
				}]
			}`),
		})
		if err != nil {
			return err
		}

		// Attach IAM policies for Lambda to access DynamoDB and CloudWatch Logs
		_, err = iam.NewRolePolicyAttachment(ctx, "lambdaDynamoPolicy", &iam.RolePolicyAttachmentArgs{
			Role:      lambdaRole.Name,
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/AmazonDynamoDBFullAccess"),
		})
		if err != nil {
			return err
		}

		_, err = iam.NewRolePolicyAttachment(ctx, "lambdaLogPolicy", &iam.RolePolicyAttachmentArgs{
			Role:      lambdaRole.Name,
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"),
		})
		if err != nil {
			return err
		}

		// Lambda function for Entry
		entryLambda, err := lambda.NewFunction(ctx, "entryHandler", &lambda.FunctionArgs{
			Role:    lambdaRole.Arn,
			Runtime: pulumi.String("provided.al2"),
			Handler: pulumi.String("bootstrap"),
			Code:    pulumi.NewFileArchive("../cmd/lambda/entry-handler.zip"),
			Environment: &lambda.FunctionEnvironmentArgs{
				Variables: pulumi.StringMap{
					"TABLE_NAME": table.Name,
				},
			},
		})
		if err != nil {
			return err
		}

		// Lambda function for Exit
		exitLambda, err := lambda.NewFunction(ctx, "exitHandler", &lambda.FunctionArgs{
			Role:    lambdaRole.Arn,
			Runtime: pulumi.String("provided.al2"),
			Handler: pulumi.String("bootstrap"),
			Code:    pulumi.NewFileArchive("../cmd/lambda/exit-handler.zip"),
			Environment: &lambda.FunctionEnvironmentArgs{
				Variables: pulumi.StringMap{
					"TABLE_NAME": table.Name,
				},
			},
		})
		if err != nil {
			return err
		}

		// API Gateway setup
		api, err := apigateway.NewRestApi(ctx, "parkingApi", &apigateway.RestApiArgs{
			Name:        pulumi.String("Parking Lot API"),
			Description: pulumi.String("API for parking lot management system"),
		})
		if err != nil {
			return err
		}

		// Create API Gateway resources for entry and exit paths
		entryResource, err := apigateway.NewResource(ctx, "entryResource", &apigateway.ResourceArgs{
			RestApi:  api.ID(),
			ParentId: api.RootResourceId,
			PathPart: pulumi.String("entry"),
		})
		if err != nil {
			return err
		}

		exitResource, err := apigateway.NewResource(ctx, "exitResource", &apigateway.ResourceArgs{
			RestApi:  api.ID(),
			ParentId: api.RootResourceId,
			PathPart: pulumi.String("exit"),
		})
		if err != nil {
			return err
		}

		// Create POST methods for each resource
		entryMethod, err := apigateway.NewMethod(ctx, "entryMethod", &apigateway.MethodArgs{
			RestApi:        api.ID(),
			ResourceId:     entryResource.ID(),
			HttpMethod:     pulumi.String("POST"),
			Authorization:  pulumi.String("NONE"),
			ApiKeyRequired: pulumi.Bool(false),
			RequestParameters: pulumi.BoolMap{
				"method.request.querystring.plate":      pulumi.Bool(true),
				"method.request.querystring.parkingLot": pulumi.Bool(true),
			},
		})
		if err != nil {
			return err
		}

		exitMethod, err := apigateway.NewMethod(ctx, "exitMethod", &apigateway.MethodArgs{
			RestApi:        api.ID(),
			ResourceId:     exitResource.ID(),
			HttpMethod:     pulumi.String("POST"),
			Authorization:  pulumi.String("NONE"),
			ApiKeyRequired: pulumi.Bool(false),
			RequestParameters: pulumi.BoolMap{
				"method.request.querystring.ticketId": pulumi.Bool(true),
			},
		})
		if err != nil {
			return err
		}

		// Add Lambda integrations
		entryIntegration, err := apigateway.NewIntegration(ctx, "entryIntegration", &apigateway.IntegrationArgs{
			RestApi:               api.ID(),
			ResourceId:            entryResource.ID(),
			HttpMethod:            entryMethod.HttpMethod,
			IntegrationHttpMethod: pulumi.String("POST"),
			Type:                  pulumi.String("AWS_PROXY"),
			Uri:                   entryLambda.InvokeArn,
		})
		if err != nil {
			return err
		}

		exitIntegration, err := apigateway.NewIntegration(ctx, "exitIntegration", &apigateway.IntegrationArgs{
			RestApi:               api.ID(),
			ResourceId:            exitResource.ID(),
			HttpMethod:            exitMethod.HttpMethod,
			IntegrationHttpMethod: pulumi.String("POST"),
			Type:                  pulumi.String("AWS_PROXY"),
			Uri:                   exitLambda.InvokeArn,
		})
		if err != nil {
			return err
		}

		// Grant API Gateway permission to invoke the Lambda functions
		_, err = lambda.NewPermission(ctx, "apiGatewayEntryPermission", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  entryLambda.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("%s/*/*/entry", api.ExecutionArn),
		})
		if err != nil {
			return err
		}

		_, err = lambda.NewPermission(ctx, "apiGatewayExitPermission", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  exitLambda.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("%s/*/*/exit", api.ExecutionArn),
		})
		if err != nil {
			return err
		}

		// Create a deployment to make the API available
		deployment, err := apigateway.NewDeployment(ctx, "apiDeployment", &apigateway.DeploymentArgs{
			RestApi:     api.ID(),
			Description: pulumi.String("Initial deployment"),
		}, pulumi.DependsOn([]pulumi.Resource{
			entryIntegration,
			exitIntegration,
		}))
		if err != nil {
			return err
		}

		// Create a stage for the deployment
		_, err = apigateway.NewStage(ctx, "prodStage", &apigateway.StageArgs{
			RestApi:    api.ID(),
			Deployment: deployment.ID(),
			StageName:  pulumi.String("prod"),
		})
		if err != nil {
			return err
		}
		// Export the API endpoint URL
		ctx.Export("apiUrl", pulumi.Sprintf("https://%s.execute-api.%s.amazonaws.com/prod", api.ID(), awsRegion))
		ctx.Export("apiEndpoint", api.ExecutionArn)
		ctx.Export("entryLambdaName", entryLambda.Name)
		ctx.Export("exitLambdaName", exitLambda.Name)
		ctx.Export("dynamoTableName", table.Name)

		return nil
	})
}
