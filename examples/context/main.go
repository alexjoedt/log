// Context example showing logger propagation through context
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/alexjoedt/log"
)

func main() {
	// Create base logger
	logger := log.New(
		log.WithLevel(log.DEBUG),
		log.WithDefaultFields("app", "context-demo"),
	)

	// Attach logger to context
	ctx := log.ContextWithLogger(context.Background(), logger)

	// Pass context through the call chain
	handleHTTPRequest(ctx, "GET", "/api/users/123")
}

func handleHTTPRequest(ctx context.Context, method, path string) {
	// Get logger from context and add request-specific fields
	logger := log.FromContext(ctx).WithFields(
		"request_id", generateRequestID(),
		"method", method,
		"path", path,
	)

	// Update context with request logger
	ctx = log.ContextWithLogger(ctx, logger)

	logger.Info("request received")

	// Call service layer
	user, err := fetchUser(ctx, 123)
	if err != nil {
		logger.Error("failed to fetch user", "error", err)
		return
	}

	logger.Info("request completed", "user", user, "duration_ms", 42)
}

func fetchUser(ctx context.Context, userID int) (string, error) {
	// Logger automatically includes request_id, method, path from context
	logger := log.FromContext(ctx)

	logger.Debug("fetching user from database", "user_id", userID)

	// Simulate database call
	time.Sleep(20 * time.Millisecond)

	// Call cache layer
	cached := checkCache(ctx, userID)
	if cached {
		logger.Debug("user found in cache", "user_id", userID)
		return fmt.Sprintf("user-%d", userID), nil
	}

	logger.Debug("user retrieved from database", "user_id", userID)
	return fmt.Sprintf("user-%d", userID), nil
}

func checkCache(ctx context.Context, userID int) bool {
	logger := log.FromContext(ctx)
	logger.Trace("checking cache", "user_id", userID)

	// Simulate cache lookup
	return userID%2 == 0
}

func generateRequestID() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}
