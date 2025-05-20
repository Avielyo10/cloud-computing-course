package lambda

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-lambda-go/events"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"parking-lot/internal/handler"
	"parking-lot/internal/logger"
	"parking-lot/internal/service"
	"parking-lot/server/api"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// Context keys
const (
	requestIDKey contextKey = "requestID"
)

// APIAdapter handles the integration with AWS Lambda
type APIAdapter struct {
	router *gin.Engine
	log    logger.Logger
}

// NewAPIAdapter creates a new API adapter for Lambda
func NewAPIAdapter() *APIAdapter {
	// Initialize logger
	log := logger.NewLogger()
	log.Info("Initializing Lambda API adapter")

	// Set Gin to release mode
	gin.SetMode(gin.ReleaseMode)

	// Create Gin router
	router := gin.New()
	router.Use(gin.Recovery())

	// Add request ID middleware
	router.Use(func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
			c.Header("X-Request-ID", requestID)
		}

		// Store the request ID in the context
		ctx := context.WithValue(c.Request.Context(), requestIDKey, requestID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	})

	// Add logging middleware
	router.Use(func(c *gin.Context) {
		reqLog := log.WithContext(c.Request.Context()).WithFields(
			logger.Field{Key: "method", Value: c.Request.Method},
			logger.Field{Key: "path", Value: c.Request.URL.Path},
			logger.Field{Key: "client_ip", Value: c.ClientIP()},
		)

		reqLog.Info("Request started")

		c.Next()

		reqLog.WithFields(
			logger.Field{Key: "status", Value: c.Writer.Status()},
		).Info("Request completed")
	})

	router.NoRoute(func(c *gin.Context) {
		c.JSON(404, api.ErrorResponse{
			Message: "Not Found",
		})
	})

	// Create service and handler
	parkingService, err := service.NewParkingLotService(context.Background())
	if err != nil {
		// Log the error and create a fallback in-memory service for development
		log.Error("Error creating DynamoDB service, falling back to in-memory",
			logger.Field{Key: "error", Value: err.Error()})
		parkingService = &service.ParkingLotService{} // Default constructor creates in-memory service
	}
	parkingHandler := handler.NewParkingHandler(parkingService)

	// Register API handlers
	api.RegisterHandlers(router, parkingHandler)

	// Create the Lambda adapter
	return &APIAdapter{
		log:    log,
		router: router,
	}
}

// ProxyWithContext handles Lambda requests
func (a *APIAdapter) ProxyWithContext(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Extract or generate a request ID
	requestID := req.Headers["X-Request-ID"]
	if requestID == "" {
		requestID = uuid.New().String()
		req.Headers["X-Request-ID"] = requestID
	}

	// Create a logger with the request ID
	reqLog := a.log.WithRequestID(requestID).WithFields(
		logger.Field{Key: "path", Value: req.Path},
		logger.Field{Key: "method", Value: req.HTTPMethod},
	)

	reqLog.Info("Lambda request received")

	// Handle the request
	adapter := ginadapter.New(a.router)
	response, err := adapter.ProxyWithContext(ctx, req)

	// Log the result
	statusCode := response.StatusCode
	reqLog.WithFields(
		logger.Field{Key: "status_code", Value: statusCode},
	).Info("Lambda request completed")

	if err != nil {
		reqLog.Error("Lambda request error", logger.Field{Key: "error", Value: err.Error()})
	}

	// Ensure the request ID is included in the response
	if response.Headers == nil {
		response.Headers = make(map[string]string)
	}
	response.Headers["X-Request-Id"] = requestID

	return response, err
}

// Cleanup performs cleanup operations for the adapter
func (a *APIAdapter) Cleanup(ctx context.Context) error {
	// Perform any necessary cleanup operations here
	// For example, closing database connections or releasing resources
	// Currently, there are no resources to clean up in this adapter
	a.log.Info("Cleaning up Lambda API adapter")
	return nil
}

// RunLocalServer starts the local server for testing with graceful shutdown
func (a *APIAdapter) RunLocalServer(ctx context.Context) {
	defer a.Cleanup(ctx)

	// Create a custom HTTP server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: a.router,
	}

	// Create a channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine
	go func() {
		a.log.Info("Starting local server on port 8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.log.Error("Failed to start local server", logger.Field{Key: "error", Value: err.Error()})
		}
	}()

	// Wait for interrupt signal
	<-quit
	a.log.Info("Shutdown signal received, gracefully stopping server...")

	// Create a deadline for shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(shutdownCtx); err != nil {
		a.log.Error("Server forced to shutdown", logger.Field{Key: "error", Value: err.Error()})
	}

	a.log.Info("Server gracefully stopped")
}
