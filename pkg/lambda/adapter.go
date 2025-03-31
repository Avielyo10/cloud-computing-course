package lambda

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"

	"parking-lot/internal/handler"
	"parking-lot/internal/service"
	"parking-lot/server/api"
)

// APIAdapter handles the integration with AWS Lambda
type APIAdapter struct {
	ginLambda *ginadapter.GinLambda
}

// NewAPIAdapter creates a new API adapter for Lambda
func NewAPIAdapter() *APIAdapter {
	// Set Gin to release mode
	gin.SetMode(gin.ReleaseMode)

	// Create Gin router
	router := gin.New()
	router.Use(gin.Recovery())

	// Create service and handler
	parkingService, err := service.NewParkingLotService()
	if err != nil {
		// Log the error and create a fallback in-memory service for development
		log.Printf("Error creating DynamoDB service: %v, falling back to in-memory", err)
		parkingService = &service.ParkingLotService{} // Default constructor creates in-memory service
	}
	parkingHandler := handler.NewParkingHandler(parkingService)

	// Register API handlers
	api.RegisterHandlers(router, parkingHandler)

	// Create the Lambda adapter
	return &APIAdapter{
		ginLambda: ginadapter.New(router),
	}
}

// ProxyWithContext handles Lambda requests
func (a *APIAdapter) ProxyWithContext(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return a.ginLambda.ProxyWithContext(ctx, req)
}
