// Testing example showing how to test code that logs
package main

import (
	"errors"
	"testing"

	"github.com/alexjoedt/log"
)

// Example function that logs
func ProcessOrder(logger *log.Logger, orderID int, valid bool) error {
	logger.Info("processing order", "order_id", orderID)

	if !valid {
		logger.Error("invalid order", "order_id", orderID, "reason", "missing required fields")
		return errors.New("invalid order")
	}

	logger.Debug("validating payment", "order_id", orderID)
	logger.Info("order processed successfully", "order_id", orderID)
	return nil
}

func TestProcessOrder_Valid(t *testing.T) {
	// Create test logger
	logger, hook := log.NewTestLogger()

	// Execute function
	err := ProcessOrder(logger.Logger, 123, true)

	// Assert no error
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Assert log entries
	if hook.Count() != 3 {
		t.Errorf("expected 3 log entries, got %d", hook.Count())
	}

	// Check specific messages
	if !hook.HasMessage("processing order") {
		t.Error("expected 'processing order' message")
	}

	if !hook.HasMessage("order processed successfully") {
		t.Error("expected 'order processed successfully' message")
	}

	// Check levels
	if hook.CountLevel(log.INFO) != 2 {
		t.Errorf("expected 2 INFO logs, got %d", hook.CountLevel(log.INFO))
	}

	if hook.CountLevel(log.DEBUG) != 1 {
		t.Errorf("expected 1 DEBUG log, got %d", hook.CountLevel(log.DEBUG))
	}

	// Check last entry
	lastEntry := hook.LastEntry()
	if lastEntry.Message != "order processed successfully" {
		t.Errorf("expected last message to be 'order processed successfully', got '%s'", lastEntry.Message)
	}
}

func TestProcessOrder_Invalid(t *testing.T) {
	// Create test logger
	logger, hook := log.NewTestLogger()

	// Execute function with invalid order
	err := ProcessOrder(logger.Logger, 456, false)

	// Assert error
	if err == nil {
		t.Error("expected error for invalid order")
	}

	// Assert error was logged
	if !hook.HasLevel(log.ERROR) {
		t.Error("expected ERROR level log")
	}

	// Check error message
	if !hook.HasMessage("invalid order") {
		t.Error("expected 'invalid order' message")
	}

	// Verify only 2 logs (processing + error, no success)
	if hook.Count() != 2 {
		t.Errorf("expected 2 log entries, got %d", hook.Count())
	}
}

func TestMultipleOrders(t *testing.T) {
	logger, hook := log.NewTestLogger()

	// Process multiple orders
	ProcessOrder(logger.Logger, 1, true)
	ProcessOrder(logger.Logger, 2, false)
	ProcessOrder(logger.Logger, 3, true)

	// Check total count
	totalLogs := hook.Count()
	if totalLogs != 8 { // 3 for first, 2 for second, 3 for third
		t.Errorf("expected 8 log entries, got %d", totalLogs)
	}

	// Reset and verify
	hook.Reset()
	if hook.Count() != 0 {
		t.Errorf("expected 0 entries after reset, got %d", hook.Count())
	}
}
