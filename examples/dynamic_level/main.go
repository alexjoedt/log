// Dynamic log level example showing runtime level changes
package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexjoedt/log"
)

func main() {
	// Create a dynamic log level
	logLevel := &slog.LevelVar{}
	logLevel.Set(slog.LevelInfo)

	// Create logger with dynamic level
	logger := log.New(
		log.WithLevel(logLevel),
		log.WithFormat(log.FormatConsole),
		log.WithDefaultFields("service", "dynamic-demo"),
	)

	// Set as default
	log.SetDefault(logger)

	log.Info("application started", "level", logLevel.Level().String())
	log.Info("current log level: INFO - debug messages will be filtered")

	// Setup signal handler to toggle debug mode
	setupSignalHandler(logLevel)

	// Simulate application work
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	count := 0
	for {
		select {
		case <-ticker.C:
			count++

			// These debug messages only appear when level is DEBUG
			log.Debug("debug message", "count", count, "timestamp", time.Now().Unix())

			// Info messages always appear (since we start at INFO)
			log.Info("processing request", "count", count)

			if count%5 == 0 {
				log.Warn("periodic health check", "status", "ok")
			}

			if count >= 20 {
				log.Info("shutting down after 20 iterations")
				return
			}
		}
	}
}

func setupSignalHandler(logLevel *slog.LevelVar) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGUSR1, syscall.SIGUSR2)

	go func() {
		for sig := range sigChan {
			switch sig {
			case syscall.SIGUSR1:
				// Toggle to DEBUG
				logLevel.Set(slog.LevelDebug)
				fmt.Println("\n🔍 Log level changed to DEBUG (send SIGUSR2 to change back to INFO)")
				log.Info("log level changed", "level", "DEBUG")

			case syscall.SIGUSR2:
				// Toggle back to INFO
				logLevel.Set(slog.LevelInfo)
				fmt.Println("\n📊 Log level changed to INFO (send SIGUSR1 to change to DEBUG)")
				log.Info("log level changed", "level", "INFO")
			}
		}
	}()

	fmt.Printf("Send signals to PID %d:\n", os.Getpid())
	fmt.Println("  kill -USR1", os.Getpid(), " → Change to DEBUG")
	fmt.Println("  kill -USR2", os.Getpid(), " → Change to INFO")
	fmt.Println()
}
