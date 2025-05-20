//go:build !integration
// +build !integration

package lambda

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"parking-lot/internal/logger"
	"parking-lot/server/api"
)

// setupTestAdapter creates a minimal APIAdapter for testing ProxyWithContext and Cleanup
func setupTestAdapter() *APIAdapter {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Middleware to set or propagate request ID
	router.Use(func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = "generated-id"
			c.Header("X-Request-ID", reqID)
		}
		ctx := context.WithValue(c.Request.Context(), requestIDKey, reqID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	// NoRoute handler matching real adapter behavior
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, api.ErrorResponse{Message: "Not Found"})
	})
	return &APIAdapter{
		router: router,
		log:    logger.NewLogger(),
	}
}

func TestRealProxyWithContext_NotFound(t *testing.T) {
	adapter := setupTestAdapter()
	req := events.APIGatewayProxyRequest{
		HTTPMethod: "GET",
		Path:       "/nope",
		Headers:    map[string]string{},
	}
	resp, err := adapter.ProxyWithContext(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Ensure response body contains the error message
	var er api.ErrorResponse
	err = json.Unmarshal([]byte(resp.Body), &er)
	assert.NoError(t, err)
	assert.Equal(t, "Not Found", er.Message)

	// Ensure request ID header is present in response
	assert.NotEmpty(t, resp.Headers["X-Request-Id"])
}

func TestRealProxyWithContext_WithRequestID(t *testing.T) {
	adapter := setupTestAdapter()
	// Provided request ID
	reqID := "test-id-456"
	req := events.APIGatewayProxyRequest{
		HTTPMethod: "GET",
		Path:       "/nope",
		Headers:    map[string]string{"X-Request-ID": reqID},
	}
	resp, err := adapter.ProxyWithContext(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	// Ensure the same request ID is returned
	assert.Equal(t, reqID, resp.Headers["X-Request-Id"])
}

func TestRealCleanup(t *testing.T) {
	adapter := setupTestAdapter()
	err := adapter.Cleanup(context.Background())
	assert.NoError(t, err)
}
