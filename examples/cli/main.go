// CLI example demonstrating CLI-optimized logging for command-line tools
package main

import (
	"fmt"
	"time"

	"github.com/alexjoedt/log"
)

func main() {
	// Create a CLI logger with default settings:
	// - Writes to stderr
	// - No timestamps
	// - Symbol prefixes enabled
	// - Clean message-only output
	logger := log.NewCLILogger()

	// Demonstrate different message types
	logger.Step("Starting deployment process")
	time.Sleep(500 * time.Millisecond)

	logger.Step("Building application")
	time.Sleep(800 * time.Millisecond)
	logger.Success("Build completed")

	logger.Step("Running tests")
	time.Sleep(600 * time.Millisecond)
	logger.Success("All tests passed")

	logger.Step("Pushing to registry")
	time.Sleep(700 * time.Millisecond)
	logger.Failure("Push failed: authentication error")

	// Standard log levels also work (but render with CLI style)
	logger.Info("Attempting retry with different credentials")
	time.Sleep(500 * time.Millisecond)

	logger.Step("Retrying push")
	time.Sleep(900 * time.Millisecond)
	logger.Success("Successfully pushed to registry")

	logger.Step("Deploying to production")
	time.Sleep(1000 * time.Millisecond)
	logger.Success("Deployment completed")

	// Warnings and errors work too
	logger.Warn("Remember to update documentation")

	// You can also create with custom options
	fmt.Println("\n--- CLI Logger with Custom Options ---")
	customLogger := log.NewCLILogger(
		log.WithCLISymbols(false), // Disable symbols
		log.WithLevel(log.DEBUG),
	)
	customLogger.Step("This step has no symbol")
	customLogger.Debug("Debug message without symbols")
	customLogger.Success("Success without symbol")

	// Or use the format option directly with New()
	fmt.Println("\n--- Using WithFormat(FormatCLI) ---")
	anotherLogger := log.New(
		log.WithFormat(log.FormatCLI),
		// Could add more options here
	)
	anotherLogger.Step("Using format option")
	anotherLogger.Success("Works the same way")
}
