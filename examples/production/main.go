// Production example with JSON logging, rotation, and hooks
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/alexjoedt/log"
)

func main() {
	// Production logger configuration
	logger := log.New().
		WithLevel(log.INFO).
		WithFormat(log.FormatJSON).
		WithDefaultFields(
			"service", "api-server",
			"version", "1.0.0",
			"environment", "production",
			"host", getHostname(),
		).
		WithCaller(). // Include caller information
		Build()

	// Set as default
	log.SetDefault(logger)

	// Register hook for error tracking
	log.RegisterHook(func(entry *log.Entry) error {
		if entry.Level >= log.ERROR {
			// In production, this would send to Sentry, DataDog, etc.
			fmt.Printf("🚨 Alert: %s - %s\n", entry.Level, entry.Message)
		}
		return nil
	})

	// Simulate application lifecycle
	log.Info("service started")

	// Simulate handling requests
	for i := 0; i < 5; i++ {
		handleRequest(i + 1)
		time.Sleep(100 * time.Millisecond)
	}

	log.Info("service stopping")
}

func handleRequest(id int) {
	// Create request-scoped logger
	reqLogger := log.WithFields(
		"request_id", fmt.Sprintf("req-%d", id),
		"method", "POST",
		"path", "/api/users",
	)

	reqLogger.Info("request received")

	// Simulate processing
	duration := time.Duration(10+id*5) * time.Millisecond
	time.Sleep(duration)

	// Simulate occasional errors
	if id%3 == 0 {
		reqLogger.Error("validation failed",
			"error", "invalid email format",
			"field", "email",
		)
		return
	}

	reqLogger.Info("request completed",
		"duration_ms", duration.Milliseconds(),
		"status", 200,
	)
}

func getHostname() string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	return hostname
}
