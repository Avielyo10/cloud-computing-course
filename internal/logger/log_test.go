package logger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLoggerCreation tests the logger creation
func TestLoggerCreation(t *testing.T) {
	// Create a new logger
	logger := NewLogger()

	// Assertion - should not be nil
	assert.NotNil(t, logger)
}

// TestLogLevels tests various log levels
func TestLogLevels(t *testing.T) {
	// Create a logger
	logger := NewLogger()

	// Test different log levels - these should not panic
	t.Run("Debug level", func(t *testing.T) {
		assert.NotPanics(t, func() {
			logger.Debug("Debug message", Field{Key: "test", Value: "value"})
		})
	})

	t.Run("Info level", func(t *testing.T) {
		assert.NotPanics(t, func() {
			logger.Info("Info message", Field{Key: "test", Value: "value"})
		})
	})

	t.Run("Warn level", func(t *testing.T) {
		assert.NotPanics(t, func() {
			logger.Warn("Warning message", Field{Key: "test", Value: "value"})
		})
	})

	t.Run("Error level", func(t *testing.T) {
		assert.NotPanics(t, func() {
			logger.Error("Error message", Field{Key: "test", Value: "value"})
		})
	})

	// Not testing Fatal because it would call os.Exit()
}

// TestWithContext tests logger with context
func TestWithContext(t *testing.T) {
	// Create a logger
	logger := NewLogger()

	// Create a context with a request ID
	ctx := context.WithValue(context.Background(), "requestID", "test-request-id")

	// Create a logger with the context
	contextLogger := logger.WithContext(ctx)

	// Assertion - should not be nil
	assert.NotNil(t, contextLogger)

	// Should not panic when logging
	assert.NotPanics(t, func() {
		contextLogger.Info("Test message with context")
	})
}

// TestWithRequestID tests logger with request ID
func TestWithRequestID(t *testing.T) {
	// Create a logger
	logger := NewLogger()

	// Create a logger with a request ID
	requestIDLogger := logger.WithRequestID("test-request-id")

	// Assertion - should not be nil
	assert.NotNil(t, requestIDLogger)

	// Should not panic when logging
	assert.NotPanics(t, func() {
		requestIDLogger.Info("Test message with request ID")
	})
}

// TestWithFields tests logger with fields
func TestWithFields(t *testing.T) {
	// Create a logger
	logger := NewLogger()

	// Create test fields
	fields := []Field{
		{Key: "string", Value: "value"},
		{Key: "number", Value: 123},
		{Key: "bool", Value: true},
	}

	// Create a logger with fields
	fieldsLogger := logger.WithFields(fields...)

	// Assertion - should not be nil
	assert.NotNil(t, fieldsLogger)

	// Should not panic when logging
	assert.NotPanics(t, func() {
		fieldsLogger.Info("Test message with fields")
	})
}
