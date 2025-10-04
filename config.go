package log

import (
	"os"
	"strconv"
	"strings"
)

// getDefaultLevel returns the default log level from environment or INFO.
func getDefaultLevel() Level {
	if envLevel := os.Getenv("LOG_LEVEL"); envLevel != "" {
		switch strings.ToUpper(envLevel) {
		case "TRACE":
			return TRACE
		case "DEBUG":
			return DEBUG
		case "INFO":
			return INFO
		case "WARN", "WARNING":
			return WARN
		case "ERROR":
			return ERROR
		case "FATAL":
			return FATAL
		}
	}
	return INFO
}

// getDefaultFormat returns the default format from environment or console.
func getDefaultFormat() Format {
	if envFormat := os.Getenv("LOG_FORMAT"); envFormat != "" {
		switch strings.ToLower(envFormat) {
		case "json":
			return FormatJSON
		case "text":
			return FormatText
		case "console":
			return FormatConsole
		}
	}
	return FormatConsole
}

// getEnvBool returns a boolean from an environment variable.
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

// getEnvInt returns an integer from an environment variable.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

// FromEnv creates a logger with configuration from environment variables.
// Supported environment variables:
//   - LOG_LEVEL: trace, debug, info, warn, error, fatal (default: info)
//   - LOG_FORMAT: json, text, console (default: console)
//   - LOG_CALLER: true/false - enable caller info (default: false)
//   - LOG_CALLER_SKIP: number - adjust caller depth (default: 2)
func FromEnv() *Logger {
	var opts []Option

	// Level is already set via getDefaultLevel() in New()
	// Format is already set via getDefaultFormat() in New()

	if getEnvBool("LOG_CALLER", false) {
		opts = append(opts, WithCaller())
	}

	callerSkip := getEnvInt("LOG_CALLER_SKIP", 2)
	if callerSkip != 2 {
		opts = append(opts, WithCallerSkip(callerSkip))
	}

	if timestampFormat := os.Getenv("LOG_TIMESTAMP_FORMAT"); timestampFormat != "" {
		opts = append(opts, WithTimestampFormat(timestampFormat))
	}

	return New(opts...)
}
