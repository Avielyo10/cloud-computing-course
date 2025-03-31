package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	lambdaAdapter "parking-lot/pkg/lambda"
)

var adapter *lambdaAdapter.APIAdapter

func init() {
	adapter = lambdaAdapter.NewAPIAdapter()
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	response, err := adapter.ProxyWithContext(ctx, req)

	// Ensure we perform cleanup on Lambda cold starts
	defer func() {
		if ctx.Err() == nil {
			adapter.Cleanup(context.Background())
		}
	}()

	return response, err
}
