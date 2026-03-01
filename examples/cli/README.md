# CLI Logger Example

This example demonstrates how to use the CLI-optimized logger for command-line applications.

## Features

The CLI logger is designed for terminal applications and provides:

- **Clean Output**: Symbol-prefixed messages without timestamps or structured fields
- **Stderr by Default**: Follows CLI conventions (diagnostics to stderr, data to stdout)
- **Auto-detected Colors**: Terminal colors when TTY is detected
- **Three Convenience Methods**:
  - `Success(msg)` - ✓ Green checkmark for successful operations
  - `Failure(msg)` - ✗ Red X for failed operations
  - `Step(msg)` - • Blue bullet for progress/steps

## Usage

### Basic Usage

```go
logger := log.NewCLILogger()

logger.Step("Building application")
logger.Success("Build completed")
logger.Failure("Push failed")
```

### With Custom Options

```go
logger := log.NewCLILogger(
    log.WithCLISymbols(false),  // Disable symbols
    log.WithLevel(log.DEBUG),    // Enable debug logs
)
```

### Using Format Option

```go
logger := log.New(
    log.WithFormat(log.FormatCLI),
    log.WithWriter(os.Stdout),  // Override default stderr
)
```

## Running the Example

```bash
go run main.go
```

## Expected Output

```
• Starting deployment process
• Building application
✓ Build completed
• Running tests
✓ All tests passed
• Pushing to registry
✗ Push failed: authentication error
• Attempting retry with different credentials
• Retrying push
✓ Successfully pushed to registry
• Deploying to production
✓ Deployment completed
⚠ Remember to update documentation
```

## Design Philosophy

CLI loggers follow POSIX conventions:
- **No timestamps** - CLI tools are typically short-lived
- **No structured fields** - Messages are human-composed strings
- **Stderr for diagnostics** - Keeps stdout clean for actual output/data
- **Symbols over text** - Visual indicators (✓✗⚠•) are more readable than [INFO] [ERROR]

## When to Use

Use CLI logger for:
- Command-line tools and utilities
- Interactive terminal applications
- Deployment scripts
- Build tools

Don't use for:
- Long-running services (use FormatJSON or FormatConsole)
- Log aggregation systems (use FormatJSON)
- When you need structured logging (use FormatJSON)
