package mocks

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/aws/aws-lambda-go/events"
	"github.com/gin-gonic/gin"

	"parking-lot/internal/logger"
)

// MockAPIAdapter is a test version of the API adapter
type MockAPIAdapter struct {
	Router *gin.Engine
	Log    logger.Logger
}

// ProxyWithContext mocks the Lambda proxy functionality for testing
func (a *MockAPIAdapter) ProxyWithContext(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Extract or use the provided request ID
	requestID := req.Headers["X-Request-ID"]
	if requestID == "" {
		requestID = "generated-id"
	}

	// Log the request
	a.Log.WithRequestID(requestID).WithFields(
		logger.Field{Key: "path", Value: req.Path},
		logger.Field{Key: "method", Value: req.HTTPMethod},
	).Info("Lambda request received")

	// Create HTTP request from API Gateway event
	httpReq, _ := http.NewRequest(req.HTTPMethod, req.Path, nil)

	// Copy headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Create response recorder
	w := httptest.NewRecorder()

	// Process the request
	a.Router.ServeHTTP(w, httpReq)

	// Convert response to API Gateway format
	resp := events.APIGatewayProxyResponse{
		StatusCode: w.Code,
		Headers:    make(map[string]string),
		Body:       w.Body.String(),
	}

	// Add headers from response
	for k, v := range w.Header() {
		if len(v) > 0 {
			resp.Headers[k] = v[0]
		}
	}

	// Ensure the request ID is included in the response
	if requestID != "" {
		resp.Headers["X-Request-Id"] = requestID
	}

	// Log the response
	a.Log.WithRequestID(requestID).WithFields(
		logger.Field{Key: "status_code", Value: resp.StatusCode},
	).Info("Lambda request completed")

	return resp, nil
}

// Cleanup performs cleanup operations
func (a *MockAPIAdapter) Cleanup(ctx context.Context) error {
	a.Log.Info("Cleaning up Lambda API adapter")
	return nil
}
