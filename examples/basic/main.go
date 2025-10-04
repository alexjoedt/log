// Basic example demonstrating simple logging usage
package main

import (
	"errors"
	"time"

	"github.com/alexjoedt/log"
)

func main() {
	// Simple package-level logging with default settings
	log.Info("application starting")
	log.Debug("this won't show with default INFO level")

	// Log with additional fields
	log.Info("user logged in", "user_id", 12345, "email", "user@example.com")

	// Log with error
	err := errors.New("connection timeout")
	log.Error("failed to connect to database", "error", err, "retry_count", 3)

	// Warnings
	log.Warn("high memory usage detected", "usage_percent", 85)

	// Create a custom logger with DEBUG level
	logger := log.New().
		WithLevel(log.DEBUG).
		Build()

	// Set as default
	log.SetDefault(logger)

	// Now debug messages will appear
	log.Debug("debug message now visible", "timestamp", time.Now())

	// Check if level is enabled to avoid expensive operations
	if log.IsDebugEnabled() {
		log.Debug("performing expensive operation", "result", expensiveOperation())
	}

	// Create a logger with context fields
	requestLogger := logger.WithFields(
		"request_id", "req-123-456",
		"method", "GET",
		"path", "/api/users",
	)

	requestLogger.Info("handling request")
	requestLogger.Info("request completed", "duration_ms", 45)

	log.Info("application stopped")
}

func expensiveOperation() string {
	// Simulate expensive operation
	time.Sleep(10 * time.Millisecond)
	return "expensive result"
}
