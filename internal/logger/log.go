package logger

import (
	"context"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Field represents a log field key-value pair
type Field struct {
	Key   string
	Value interface{}
}

// Logger defines the logging interface for the application
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	WithContext(ctx context.Context) Logger
	WithRequestID(requestID string) Logger
	WithFields(fields ...Field) Logger
}

type zerologLogger struct {
	log zerolog.Logger
}

// NewLogger creates a new logger instance
func NewLogger() Logger {
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}

	// Set a more readable format for local development
	if os.Getenv("AWS_EXECUTION_ENV") == "" {
		output.FormatLevel = func(i interface{}) string {
			return zerolog.LevelFieldMarshalFunc(i.(zerolog.Level))
		}
	}

	logger := zerolog.New(output).
		With().
		Timestamp().
		Caller().
		Logger()

	return &zerologLogger{log: logger}
}

func (l *zerologLogger) Debug(msg string, fields ...Field) {
	l.logWithLevel(zerolog.DebugLevel, msg, fields...)
}

func (l *zerologLogger) Info(msg string, fields ...Field) {
	l.logWithLevel(zerolog.InfoLevel, msg, fields...)
}

func (l *zerologLogger) Warn(msg string, fields ...Field) {
	l.logWithLevel(zerolog.WarnLevel, msg, fields...)
}

func (l *zerologLogger) Error(msg string, fields ...Field) {
	l.logWithLevel(zerolog.ErrorLevel, msg, fields...)
}

func (l *zerologLogger) Fatal(msg string, fields ...Field) {
	l.logWithLevel(zerolog.FatalLevel, msg, fields...)
}

func (l *zerologLogger) WithContext(ctx context.Context) Logger {
	// Extract request ID from context if available
	requestID, _ := ctx.Value("requestID").(string)
	if requestID == "" {
		requestID = uuid.New().String()
	}

	newLogger := l.log.With().Str("request_id", requestID).Logger()
	return &zerologLogger{log: newLogger}
}

func (l *zerologLogger) WithRequestID(requestID string) Logger {
	newLogger := l.log.With().Str("request_id", requestID).Logger()
	return &zerologLogger{log: newLogger}
}

func (l *zerologLogger) WithFields(fields ...Field) Logger {
	ctx := l.log.With()
	for _, field := range fields {
		ctx = ctx.Interface(field.Key, field.Value)
	}
	return &zerologLogger{log: ctx.Logger()}
}

func (l *zerologLogger) logWithLevel(level zerolog.Level, msg string, fields ...Field) {
	event := l.log.WithLevel(level)
	for _, field := range fields {
		event = event.Interface(field.Key, field.Value)
	}
	event.Msg(msg)
}
