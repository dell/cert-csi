package utils

import (
	"context"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestGetLoggerFromContext(t *testing.T) {
	// Test case: Context with logger
	t.Run("WithLogger", func(t *testing.T) {
		logger := logrus.NewEntry(logrus.StandardLogger())
		ctx := context.WithValue(context.Background(), LoggerContextKey, logger)
		result := GetLoggerFromContext(ctx)
		if result != logger {
			t.Errorf("Expected %v, got %v", logger, result)
		}
	})
	// Test case: Context without logger
	t.Run("WithoutLogger", func(t *testing.T) {
		ctx := context.Background()
		result := GetLoggerFromContext(ctx)
		if result == nil {
			t.Error("Expected non-nil logger, got nil")
		}
	})
	// Test case: Context with non-logger value
	t.Run("WithNonLoggerValue", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), LoggerContextKey, "not a logger")
		result := GetLoggerFromContext(ctx)
		if result == nil {
			t.Error("Expected non-nil logger, got nil")
		}
	})
}
