# slog Handler Example

This example demonstrates how to use `log.NewSlogHandler()` to create a fully-configured `slog.Handler` that can be used directly with Go's native `slog.Logger`.

## Use Cases

Use `NewSlogHandler()` when:
- You prefer working directly with `slog.Logger`
- You need to pass a handler to third-party libraries that expect `slog.Handler`
- You want to use this package's features (rotation, sampling, console formatting, hooks) in a pure slog workflow

## Features Demonstrated

- Creating handlers with `NewSlogHandler()`
- Using enhanced handlers with `slog.New()`
- JSON and console output formats
- Default fields applied to the handler
- All features work identically to `log.New()`

## Running

```bash
go run main.go
```

## Key Differences

### Using log.New() (Logger wrapper)
```go
logger := log.New(
    log.WithLevel(log.DEBUG),
    log.WithDefaultFields("service", "api"),
)
logger.Info("message")
```

### Using log.NewSlogHandler() (Direct slog)
```go
handler := log.NewSlogHandler(
    log.WithLevel(log.DEBUG),
    log.WithDefaultFields("service", "api"),
)
logger := slog.New(handler)
logger.Info("message")
```

Both approaches provide identical functionality - choose based on your workflow preference.
