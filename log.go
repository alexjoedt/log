// Package log provides a production-ready logging package that wraps Go's slog
// with enhanced features including beautiful console output, log rotation, sampling,
// and more.
//
// Basic usage with the logger wrapper:
//
//	logger := log.New(
//	    log.WithLevel(log.INFO),
//	    log.WithFormat(log.FormatJSON),
//	)
//	logger.Info("hello world", "key", "value")
//
// Or use the default logger directly:
//
//	log.Info("hello world")
//
// For slog-first workflows, create a handler directly:
//
//	handler := log.NewSlogHandler(
//	    log.WithLevel(log.DEBUG),
//	    log.WithFormat(log.FormatJSON),
//	    log.WithDefaultFields("service", "api"),
//	)
//	logger := slog.New(handler)
//	logger.Info("using slog with enhanced features")
package log

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

// Level represents the severity level of a log message.
type Level int

const (
	// TRACE is for very detailed trace information
	TRACE Level = iota - 8
	// DEBUG is for debug information
	DEBUG Level = iota - 4
	// INFO is for informational messages
	INFO Level = 0
	// WARN is for warning messages
	WARN Level = 4
	// ERROR is for error messages
	ERROR Level = 8
	// FATAL is for fatal errors (logs and exits)
	FATAL Level = 12
)

// String returns the string representation of the level.
func (l Level) String() string {
	switch l {
	case TRACE:
		return "TRACE"
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// ToSlogLevel converts our Level to slog.Level
func (l Level) ToSlogLevel() slog.Level {
	return slog.Level(l)
}

// Format represents the output format for logs.
type Format string

const (
	// FormatConsole is human-readable format with colors (when TTY detected)
	FormatConsole Format = "console"
	// FormatJSON is structured JSON output
	FormatJSON Format = "json"
	// FormatText is plain key=value format
	FormatText Format = "text"
)

var (
	// defaultLogger is the package-level logger
	defaultLogger     *Logger
	defaultLoggerLock sync.RWMutex

	// exitHandler is called by Fatal() - defaults to os.Exit(1)
	exitHandler     func(int)
	exitHandlerLock sync.RWMutex
)

func init() {
	// Initialize default logger with sensible defaults
	defaultLogger = New()
	exitHandler = func(code int) {
		os.Exit(code)
	}
}

// SetDefault sets the default logger used by package-level functions.
func SetDefault(logger *Logger) {
	defaultLoggerLock.Lock()
	defer defaultLoggerLock.Unlock()
	defaultLogger = logger
}

// Default returns the current default logger.
func Default() *Logger {
	defaultLoggerLock.RLock()
	defer defaultLoggerLock.RUnlock()
	return defaultLogger
}

// SetExitHandler sets the function called by Fatal().
// The default handler calls os.Exit(1).
func SetExitHandler(handler func(int)) {
	exitHandlerLock.Lock()
	defer exitHandlerLock.Unlock()
	exitHandler = handler
}

// getExitHandler returns the current exit handler.
func getExitHandler() func(int) {
	exitHandlerLock.RLock()
	defer exitHandlerLock.RUnlock()
	return exitHandler
}

// Package-level logging functions using the default logger

// Trace logs at TRACE level with the default logger.
func Trace(msg string, args ...any) {
	Default().Trace(msg, args...)
}

// Debug logs at DEBUG level with the default logger.
func Debug(msg string, args ...any) {
	Default().Debug(msg, args...)
}

// Info logs at INFO level with the default logger.
func Info(msg string, args ...any) {
	Default().Info(msg, args...)
}

// Warn logs at WARN level with the default logger.
func Warn(msg string, args ...any) {
	Default().Warn(msg, args...)
}

// Error logs at ERROR level with the default logger.
func Error(msg string, args ...any) {
	Default().Error(msg, args...)
}

// Fatal logs at FATAL level with the default logger and then exits.
func Fatal(msg string, args ...any) {
	Default().Fatal(msg, args...)
}

// IsTraceEnabled returns true if TRACE level is enabled.
func IsTraceEnabled() bool {
	return Default().IsLevelEnabled(TRACE)
}

// IsDebugEnabled returns true if DEBUG level is enabled.
func IsDebugEnabled() bool {
	return Default().IsLevelEnabled(DEBUG)
}

// IsInfoEnabled returns true if INFO level is enabled.
func IsInfoEnabled() bool {
	return Default().IsLevelEnabled(INFO)
}

// IsWarnEnabled returns true if WARN level is enabled.
func IsWarnEnabled() bool {
	return Default().IsLevelEnabled(WARN)
}

// IsErrorEnabled returns true if ERROR level is enabled.
func IsErrorEnabled() bool {
	return Default().IsLevelEnabled(ERROR)
}

// WithFields creates a new logger from the default logger with additional fields.
func WithFields(args ...any) *Logger {
	return Default().WithFields(args...)
}

// Writer returns an io.Writer that writes to the logger at the given level.
func Writer(level Level) io.Writer {
	return Default().Writer(level)
}

// LazyFunc is a function that computes a value lazily.
type LazyFunc func() any

// Lazy creates a lazy-evaluated value for expensive operations.
// The function is only called if the log level is enabled.
func Lazy(fn LazyFunc) slog.Attr {
	return slog.Any("", &lazyValue{fn: fn})
}

type lazyValue struct {
	fn LazyFunc
}

func (l *lazyValue) LogValue() slog.Value {
	return slog.AnyValue(l.fn())
}

// Entry represents a log entry, used for hooks.
type Entry struct {
	Level   Level
	Message string
	Time    interface{}
	Fields  []any
	Error   error
}

// Hook is a function called for each log entry.
// Returning an error prevents the log from being written.
type Hook func(*Entry) error

var (
	globalHooks     []Hook
	globalHooksLock sync.RWMutex
)

// RegisterHook registers a global hook that is called for all loggers.
func RegisterHook(hook Hook) {
	globalHooksLock.Lock()
	defer globalHooksLock.Unlock()
	globalHooks = append(globalHooks, hook)
}

// getGlobalHooks returns a copy of global hooks.
func getGlobalHooks() []Hook {
	globalHooksLock.RLock()
	defer globalHooksLock.RUnlock()
	hooks := make([]Hook, len(globalHooks))
	copy(hooks, globalHooks)
	return hooks
}

// ContextWithLogger returns a new context with the logger attached.
func ContextWithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

type loggerContextKey struct{}

// FromContext retrieves the logger from the context.
// If no logger is found, returns the default logger.
func FromContext(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(loggerContextKey{}).(*Logger); ok {
		return logger
	}
	return Default()
}

// ParseLogLevel parses a string and returns the corresponding log Level.
// It accepts common level abbreviations and full names (case-insensitive).
// Leading and trailing whitespace is ignored.
//
// Supported values:
//   - "trace", "trc" -> TRACE
//   - "debug", "dbg" -> DEBUG
//   - "info", "inf" -> INFO
//   - "warn", "warning" -> WARN
//   - "error", "err" -> ERROR
//   - "fatal", "fat", "ftl" -> FATAL
//
// If the input doesn't match any recognized level, it returns ERROR as the default.
func ParseLogLevel(levelStr string) Level {
	normalized := strings.ToLower(strings.TrimSpace(levelStr))

	switch normalized {
	case "trace", "trc":
		return TRACE
	case "debug", "dbg":
		return DEBUG
	case "info", "inf":
		return INFO
	case "warning", "warn":
		return WARN
	case "error", "err":
		return ERROR
	case "fatal", "fat", "fatl", "ftl":
		return FATAL
	default:
		return ERROR
	}
}
