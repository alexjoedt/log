# log

[![Go Reference](https://pkg.go.dev/badge/github.com/alexjoedt/log.svg)](https://pkg.go.dev/github.com/alexjoedt/log)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexjoedt/log)](https://goreportcard.com/report/github.com/alexjoedt/log)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A production-ready logging package for Go that wraps `slog` (Go 1.21+) with enhanced features and a developer-friendly API.

## Features

✨ **Beautiful Console Output** - Auto-detects TTY with color-coded log levels  
📊 **Multiple Formats** - Console, JSON, and Text formats  
🎯 **Six Log Levels** - TRACE, DEBUG, INFO, WARN, ERROR, FATAL  
🔧 **Functional Options API** - Intuitive configuration with option functions  
🔄 **Log Rotation** - Automatic rotation by size, age, and backup count  
📦 **Buffered Writing** - Configurable buffering with auto-flush  
🎣 **Hooks System** - Pre-log hooks for integration (Sentry, metrics, etc.)  
🎲 **Sampling** - Control log volume with smart sampling  
🧪 **Testing Support** - Built-in test logger with capture hooks  
🚀 **High Performance** - Minimal overhead over raw slog  
🔌 **Context Integration** - First-class context.Context support  
🌍 **Environment Config** - Configure via environment variables  

## Installation

```bash
go get github.com/alexjoedt/log
```

Requires Go 1.23 or later.

## Quick Start

```go
package main

import "github.com/alexjoedt/log"

func main() {
    // Use package-level functions with sensible defaults
    log.Info("application started")
    log.Debug("processing", "count", 42)
    log.Error("something failed", "error", err)
}
```

## Usage

### Basic Logging

```go
// Package-level functions (uses default logger)
log.Trace("very detailed trace")
log.Debug("debug information")
log.Info("informational message")
log.Warn("warning message")
log.Error("error occurred", "error", err)
log.Fatal("critical failure") // logs and exits with code 1
```

### Creating Custom Loggers

```go
// Create a custom logger with options
logger := log.New(
    log.WithLevel(log.DEBUG),
    log.WithFormat(log.FormatJSON),
    log.WithDefaultFields("service", "api", "version", "1.0.0"),
)

// Set as default
log.SetDefault(logger)
```

### Log Levels

The package supports six log levels in order of severity:

| Level | Value | Description |
|-------|-------|-------------|
| TRACE | -8 | Very detailed trace information |
| DEBUG | -4 | Debug information for developers |
| INFO | 0 | General informational messages |
| WARN | 4 | Warning messages |
| ERROR | 8 | Error conditions |
| FATAL | 12 | Fatal errors (logs and exits) |

```go
// Check if level is enabled (avoid expensive operations)
if log.IsDebugEnabled() {
    log.Debug("expensive data", "dump", computeExpensiveValue())
}
```

### Output Formats

#### Console Format (default)

Human-readable format with color-coded levels (when TTY detected):

```
2024-10-03T10:15:30Z [INFO] application started service=api version=1.0
2024-10-03T10:15:31Z [ERROR] database connection failed error="connection timeout"
```

#### JSON Format

Structured JSON for production and log aggregators:

```json
{"time":"2024-10-03T10:15:30Z","level":"INFO","msg":"application started","service":"api","version":"1.0"}
{"time":"2024-10-03T10:15:31Z","level":"ERROR","msg":"database failed","error":"connection timeout"}
```

#### Text Format

Plain key=value format:

```
time=2024-10-03T10:15:30Z level=INFO msg="application started" service=api version=1.0
```

### Adding Context Fields

```go
// Create logger with default fields
logger := log.New(
    log.WithDefaultFields("app", "myapp", "env", "prod"),
)

// Add fields to specific logger instance (immutable pattern)
requestLogger := logger.WithFields(
    "request_id", uuid.New(),
    "user_id", 12345,
)

requestLogger.Info("processing request")
// Output includes: app=myapp env=prod request_id=... user_id=12345
```

### Timestamp Configuration

By default, all log formats include timestamps in RFC3339 format. You can customize or disable timestamps:

```go
// Custom timestamp format
logger := log.New(
    log.WithTimestampFormat(time.RFC3339Nano),
)

// Disable timestamps completely (works for all formats)
logger := log.New(
    log.WithoutTimestamp(),
)
```

**Example output without timestamps:**

```
Console: [INFO ] application started service=api
JSON:    {"level":"INFO","msg":"application started","service":"api"}
Text:    level=INFO msg="application started" service=api
```

### Configuration Options

#### Option Functions

```go
logger := log.New(
    log.WithLevel(log.DEBUG),                    // Set minimum log level
    log.WithFormat(log.FormatJSON),              // Set output format
    log.WithWriter(os.Stdout),                   // Set output writer
    log.WithTimestampFormat(time.RFC3339Nano),   // Custom timestamp format
    log.WithoutTimestamp(),                      // Disable timestamp output
    log.WithCaller(),                            // Enable caller info (file:line)
    log.WithCallerSkip(2),                       // Adjust caller depth
    log.WithDefaultFields("key", "value"),       // Add default fields
)
```

#### Environment Variables

Configure the default logger via environment variables:

```bash
export LOG_LEVEL=debug              # trace, debug, info, warn, error, fatal
export LOG_FORMAT=json              # console, json, text
export LOG_CALLER=true              # Enable caller information
export LOG_CALLER_SKIP=2            # Adjust caller depth
export LOG_TIMESTAMP_FORMAT=RFC3339 # Timestamp format
```

```go
// Create logger from environment
logger := log.FromEnv()
```

### Advanced Features

#### Log Rotation

Automatically rotate logs based on size, age, and backup count:

```go
logger := log.New(
    log.WithRotation(
        100,  // MaxSize: 100 MB
        7,    // MaxBackups: keep 7 old files
        30,   // MaxAge: 30 days
        true, // Compress: gzip old files
    ),
)
```

#### Sampling

Reduce log volume by sampling (useful for high-frequency logs):

```go
logger := log.New(
    log.WithSampling(
        100,  // Log first 100 messages
        100,  // Then log every 100th message
    ),
)
```

#### Buffered Writing

Buffer logs for better performance with periodic flushing:

```go
logger := log.New(
    log.WithBuffer(
        8192,                    // 8KB buffer
        100*time.Millisecond,    // Flush every 100ms
    ),
)
// Automatically flushes on ERROR and FATAL
```

#### Multiple Writers (Fan-out)

Write logs to multiple destinations simultaneously:

```go
logger := log.New(
    log.WithWriters(
        os.Stdout,
        fileWriter,
        networkWriter,
    ),
)
```

#### Hooks

Register hooks to intercept log entries (e.g., send errors to Sentry):

```go
// Global hook (affects all loggers)
log.RegisterHook(func(entry *log.Entry) error {
    if entry.Level >= log.ERROR {
        sentry.CaptureException(entry.Error)
    }
    return nil
})

// Per-logger hook
logger := log.New(
    log.WithHook(func(entry *log.Entry) error {
        metrics.IncrementCounter("logs", entry.Level.String())
        return nil
    }),
)
```

#### Context Integration

Store and retrieve loggers from context:

```go
// Add logger to context
ctx := log.ContextWithLogger(ctx, logger)

// Retrieve and use logger from context
log.FromContext(ctx).Info("message with context logger")

// Falls back to default logger if none in context
log.FromContext(context.Background()).Info("uses default")
```

#### Lazy Evaluation

Avoid expensive operations when log level is disabled:

```go
log.Debug("data dump", log.Lazy(func() interface{} {
    return expensiveSerialize(data)
}))
// Function only called if DEBUG level is enabled
```

### Testing Support

Built-in testing utilities for unit tests:

```go
func TestMyFunction(t *testing.T) {
    // Create test logger with hook
    logger, hook := log.NewTestLogger()
    
    // Use logger in your code
    myFunction(logger)
    
    // Assert on captured logs
    assert.Equal(t, 3, hook.Count())
    assert.True(t, hook.HasMessage("expected message"))
    assert.True(t, hook.HasLevel(log.ERROR))
    assert.Equal(t, "last message", hook.LastEntry().Message)
    
    // Check specific level counts
    assert.Equal(t, 1, hook.CountLevel(log.ERROR))
    
    // Reset for next test
    hook.Reset()
}
```

### Custom Exit Handler

Customize behavior for FATAL logs:

```go
log.SetExitHandler(func(code int) {
    cleanup()
    os.Exit(code)
})
```

### Slog Compatibility

#### Direct slog.Handler Usage

For slog-first workflows, create a handler directly and use it with `slog.New()`:

```go
handler := log.NewSlogHandler(
    log.WithLevel(log.DEBUG),
    log.WithFormat(log.FormatJSON),
    log.WithDefaultFields("service", "api", "version", "1.0"),
)

logger := slog.New(handler)
logger.Info("using slog with enhanced features")
```

All options work with `NewSlogHandler()`:
- Log levels, formats, and writers
- Rotation, sampling, and buffering
- Hooks and caller information
- Default fields (applied via `handler.WithAttrs()`)

This is useful when:
- You prefer working directly with `slog.Logger`
- You need to pass a handler to third-party libraries
- You want a pure slog workflow with enhanced features

#### Access Underlying Logger

Access the underlying `*slog.Logger` for compatibility:

```go
logger := log.New()
slogLogger := logger.Slog()
// Use with any library that expects *slog.Logger
```

## Performance

Benchmarks show minimal overhead compared to raw slog:

```
BenchmarkLoggerInfo-8        2920038    401.2 ns/op    1363 B/op    5 allocs/op
BenchmarkSlogInfo-8          2310099    524.9 ns/op     240 B/op    0 allocs/op
BenchmarkLoggerDisabled-8   38244826     29.9 ns/op       8 B/op    0 allocs/op
```

## Examples

See the [examples](examples/) directory for complete working examples:

- [Basic usage](examples/basic/)
- [Production setup](examples/production/)
- [Direct slog.Handler](examples/slog_handler/)
- [Testing](examples/testing/)
- [Hooks integration](examples/hooks/)
- [Context usage](examples/context/)

## Best Practices

1. **Use package-level functions for simple cases**: `log.Info()`, `log.Error()`, etc.
2. **Create custom loggers for services**: Configure once with options, use throughout the service
3. **Add context fields**: Use `WithFields()` to add request IDs, user IDs, etc.
4. **Check levels for expensive operations**: Use `IsDebugEnabled()` before expensive computations
5. **Use JSON format in production**: Better for log aggregation and parsing
6. **Enable caller info for debugging**: Use `WithCaller()` option during development
7. **Set up log rotation**: Prevent disk space issues in production
8. **Use hooks for integrations**: Integrate with Sentry, metrics, etc.
9. **Test with TestLogger**: Capture and assert on logs in unit tests
10. **Configure via environment**: Use `FromEnv()` for 12-factor app compliance

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

Built on top of Go's excellent `log/slog` package introduced in Go 1.21.

