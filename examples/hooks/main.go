// Hooks example showing integration with external services
package main

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/alexjoedt/log"
)

// Simulated metrics collector
type MetricsCollector struct {
	logCount   atomic.Int64
	errorCount atomic.Int64
}

func (m *MetricsCollector) IncrementLogs(level log.Level) {
	m.logCount.Add(1)
	if level >= log.ERROR {
		m.errorCount.Add(1)
	}
}

func (m *MetricsCollector) GetStats() (logs, errors int64) {
	return m.logCount.Load(), m.errorCount.Load()
}

// Simulated error tracking service (like Sentry)
type ErrorTracker struct {
	errors []string
}

func (e *ErrorTracker) CaptureError(message string, level log.Level) {
	e.errors = append(e.errors, fmt.Sprintf("[%s] %s", level, message))
}

func (e *ErrorTracker) GetErrors() []string {
	return e.errors
}

func main() {
	// Create metrics collector and error tracker
	metrics := &MetricsCollector{}
	errorTracker := &ErrorTracker{}

	// Create logger with hooks
	logger := log.New(
		log.WithLevel(log.DEBUG),
		log.WithHook(createMetricsHook(metrics)),
		log.WithHook(createErrorTrackingHook(errorTracker)),
	)

	log.SetDefault(logger)

	// Simulate application activity
	log.Info("application started")
	log.Debug("initializing components")

	// Simulate some operations
	for i := 0; i < 10; i++ {
		processTask(i)
		time.Sleep(50 * time.Millisecond)
	}

	log.Info("application stopped")

	// Print metrics
	logs, errors := metrics.GetStats()
	fmt.Printf("\n=== Metrics ===\n")
	fmt.Printf("Total logs: %d\n", logs)
	fmt.Printf("Error logs: %d\n", errors)

	// Print captured errors
	fmt.Printf("\n=== Error Tracker ===\n")
	trackedErrors := errorTracker.GetErrors()
	if len(trackedErrors) == 0 {
		fmt.Println("No errors tracked")
	} else {
		for _, err := range trackedErrors {
			fmt.Printf("- %s\n", err)
		}
	}
}

func processTask(id int) {
	logger := log.WithFields("task_id", id)

	logger.Debug("processing task")

	// Simulate occasional errors
	if id%3 == 0 {
		logger.Error("task failed", "reason", "simulated error")
		return
	}

	// Simulate warnings
	if id%4 == 0 {
		logger.Warn("task took longer than expected", "duration_ms", 150)
	}

	logger.Info("task completed successfully")
}

// createMetricsHook creates a hook that tracks log metrics
func createMetricsHook(metrics *MetricsCollector) log.Hook {
	return func(entry *log.Entry) error {
		metrics.IncrementLogs(entry.Level)
		return nil
	}
}

// createErrorTrackingHook creates a hook that captures errors
func createErrorTrackingHook(tracker *ErrorTracker) log.Hook {
	return func(entry *log.Entry) error {
		if entry.Level >= log.ERROR {
			tracker.CaptureError(entry.Message, entry.Level)
		}
		return nil
	}
}
