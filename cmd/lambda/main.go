package main

import (
	"github.com/aws/aws-lambda-go/lambda"

	lambdaAdapter "parking-lot/pkg/lambda"
)

var adapter *lambdaAdapter.APIAdapter

func init() {
	adapter = lambdaAdapter.NewAPIAdapter()
}

func main() {
	lambda.Start(adapter.ProxyWithContext)
}
