# Examples

This directory contains complete working examples demonstrating various features of the log package.

## Running Examples

```bash
# Basic usage
cd basic && go run main.go

# Production setup with JSON logging and hooks
cd production && go run main.go

# Context-based logging
cd context && go run main.go

# Hooks integration with metrics and error tracking
cd hooks && go run main.go

# Testing (run tests)
cd testing && go test -v
```

## Examples Overview

### [basic/](basic/) - Basic Usage
Demonstrates:
- Simple package-level logging
- Creating custom loggers
- Adding fields to logs
- Checking log levels
- Logger chaining with `WithFields()`

### [production/](production/) - Production Setup
Demonstrates:
- JSON format for production
- Default fields (service, version, environment)
- Caller information
- Global hooks for error tracking
- Request-scoped loggers

### [context/](context/) - Context Integration
Demonstrates:
- Attaching loggers to context
- Propagating loggers through call chains
- Automatic field inheritance
- Request tracing

### [hooks/](hooks/) - Hooks Integration
Demonstrates:
- Creating custom hooks
- Metrics collection
- Error tracking integration
- Multiple hooks on same logger

### [testing/](testing/) - Testing Support
Demonstrates:
- Using `NewTestLogger()` in tests
- Capturing log entries
- Asserting on log messages and levels
- Counting logs by level
- Resetting captured logs

## Additional Features

For examples of features not covered here, see the main [README.md](../README.md):
- Log rotation
- Sampling
- Buffered writing
- Multiple writers
- Lazy evaluation
- Environment configuration
