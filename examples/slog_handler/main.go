package main

import (
	"log/slog"
	"os"

	"github.com/alexjoedt/log"
)

func main() {
	// Create handler with log package features
	handler := log.NewSlogHandler(
		log.WithLevel(log.DEBUG),
		log.WithFormat(log.FormatJSON),
		log.WithWriter(os.Stdout),
		log.WithDefaultFields("service", "example", "version", "1.0"),
	)

	// Use with slog directly
	logger := slog.New(handler)
	logger.Info("using slog with enhanced handler")
	logger.Debug("debug message", "key", "value")
	logger.Warn("warning message", "user_id", 12345)

	// Can also use with console format
	consoleHandler := log.NewSlogHandler(
		log.WithLevel(log.INFO),
		log.WithFormat(log.FormatConsole),
		log.WithWriter(os.Stdout),
		log.WithDefaultFields("app", "demo"),
	)

	consoleLogger := slog.New(consoleHandler)
	consoleLogger.Info("console formatted message", "status", "ok")
	consoleLogger.Error("error message", "error", "something went wrong")
}
