package main

import (
	"context"

	lambdaAdapter "parking-lot/pkg/lambda"
)

func main() {
	ctx := context.Background()
	adapter := lambdaAdapter.NewAPIAdapter()
	adapter.RunLocalServer(ctx)
}
