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
		// DynamoDB table definition matching Go data model
		table, err := dynamodb.NewTable(ctx, "parkingTickets", &dynamodb.TableArgs{
			Attributes: dynamodb.TableAttributeArray{
				&dynamodb.TableAttributeArgs{Name: pulumi.String("ticketId"), Type: pulumi.String("S")},
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
			Runtime: pulumi.String("go1.x"),
			Handler: pulumi.String("bootstrap"),
			Code:    pulumi.NewFileArchive("./cmd/lambda/entry-handler.zip"),
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
			Runtime: pulumi.String("go1.x"),
			Handler: pulumi.String("bootstrap"),
			Code:    pulumi.NewFileArchive("./cmd/lambda/exit-handler.zip"),
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
		api, err := apigateway.NewRestApi(ctx, "parkingApi", &apigateway.RestApiArgs{})
		if err != nil {
			return err
		}

		// (Optional) Integrate Lambdas with API Gateway here.

		ctx.Export("apiEndpoint", api.ExecutionArn)
		ctx.Export("entryLambdaName", entryLambda.Name)
		ctx.Export("exitLambdaName", exitLambda.Name)

		return nil
	})
}
