package log

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"
)

// Logger is the main logging type that wraps slog.Logger with enhanced features.
type Logger struct {
	slog   *slog.Logger
	level  Level
	format Format
	writer io.Writer
	fields []any
	hooks  []Hook
	config *Config
	mu     sync.RWMutex
}

// Config holds the configuration for a Logger.
type Config struct {
	Level            Level
	Format           Format
	Writer           io.Writer
	TimestampFormat  string
	ShowCaller       bool
	CallerSkip       int
	DefaultFields    []any
	EnableStackTrace Level // Show stack trace for this level and above

	// Rotation settings
	Rotation *RotationConfig

	// Sampling settings
	Sampling *SamplingConfig

	// Buffer settings
	Buffer *BufferConfig

	// Multiple writers
	Writers []io.Writer

	// Hooks
	Hooks []Hook
}

// RotationConfig holds log rotation settings.
type RotationConfig struct {
	MaxSize    int  // Maximum size in MB before rotation
	MaxBackups int  // Maximum number of old log files to retain
	MaxAge     int  // Maximum days to retain old log files
	Compress   bool // Whether to compress rotated files
}

// SamplingConfig holds sampling settings.
type SamplingConfig struct {
	First      int // Log first N messages
	Thereafter int // Then log every Nth message
}

// BufferConfig holds buffering settings.
type BufferConfig struct {
	Size          int           // Buffer size in bytes
	FlushInterval time.Duration // Auto-flush interval
}

// Builder provides a fluent API for creating loggers.
type Builder struct {
	config *Config
}

// New creates a new Logger builder with default settings.
func New() *Builder {
	return &Builder{
		config: &Config{
			Level:           getDefaultLevel(),
			Format:          getDefaultFormat(),
			Writer:          os.Stderr,
			TimestampFormat: time.RFC3339,
			CallerSkip:      2,
			DefaultFields:   []any{},
			Hooks:           []Hook{},
		},
	}
}

// WithLevel sets the minimum log level.
func (b *Builder) WithLevel(level Level) *Builder {
	b.config.Level = level
	return b
}

// WithFormat sets the output format.
func (b *Builder) WithFormat(format Format) *Builder {
	b.config.Format = format
	return b
}

// WithWriter sets the output writer.
func (b *Builder) WithWriter(w io.Writer) *Builder {
	b.config.Writer = w
	return b
}

// WithWriters sets multiple output writers (fan-out).
func (b *Builder) WithWriters(writers ...io.Writer) *Builder {
	b.config.Writers = writers
	return b
}

// WithTimestampFormat sets the timestamp format.
func (b *Builder) WithTimestampFormat(format string) *Builder {
	b.config.TimestampFormat = format
	return b
}

// WithCaller enables or disables caller information.
func (b *Builder) WithCaller() *Builder {
	b.config.ShowCaller = true
	return b
}

// WithCallerSkip sets the number of stack frames to skip for caller info.
func (b *Builder) WithCallerSkip(skip int) *Builder {
	b.config.CallerSkip = skip
	return b
}

// WithFields adds default fields to all log messages.
func (b *Builder) WithDefaultFields(args ...any) *Builder {
	b.config.DefaultFields = append(b.config.DefaultFields, args...)
	return b
}

// WithStackTrace enables stack traces for the given level and above.
func (b *Builder) WithStackTrace(level Level) *Builder {
	b.config.EnableStackTrace = level
	return b
}

// WithRotation configures log rotation.
func (b *Builder) WithRotation(maxSize, maxBackups, maxAge int, compress bool) *Builder {
	b.config.Rotation = &RotationConfig{
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   compress,
	}
	return b
}

// WithSampling configures log sampling.
func (b *Builder) WithSampling(first, thereafter int) *Builder {
	b.config.Sampling = &SamplingConfig{
		First:      first,
		Thereafter: thereafter,
	}
	return b
}

// WithBuffer configures buffered writing.
func (b *Builder) WithBuffer(size int, flushInterval time.Duration) *Builder {
	b.config.Buffer = &BufferConfig{
		Size:          size,
		FlushInterval: flushInterval,
	}
	return b
}

// WithHook adds a hook to this logger.
func (b *Builder) WithHook(hook Hook) *Builder {
	b.config.Hooks = append(b.config.Hooks, hook)
	return b
}

// Build creates the Logger from the builder configuration.
func (b *Builder) Build() *Logger {
	config := b.config

	// Determine the writer
	var writer io.Writer
	if len(config.Writers) > 1 {
		writer = &multiWriter{writers: config.Writers}
	} else if len(config.Writers) == 1 {
		writer = config.Writers[0]
	} else {
		writer = config.Writer
	}

	// Apply rotation if configured
	if config.Rotation != nil {
		writer = newRotatingWriter(writer, config.Rotation)
	}

	// Apply sampling if configured
	if config.Sampling != nil {
		writer = newSamplingWriter(writer, config.Sampling)
	}

	// Apply buffering if configured
	if config.Buffer != nil {
		writer = newBufferedWriter(writer, config.Buffer)
	}

	// Create the appropriate handler
	var handler slog.Handler
	switch config.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level:     config.Level.ToSlogLevel(),
			AddSource: config.ShowCaller,
		})
	case FormatText:
		handler = slog.NewTextHandler(writer, &slog.HandlerOptions{
			Level:     config.Level.ToSlogLevel(),
			AddSource: config.ShowCaller,
		})
	case FormatConsole:
		handler = newConsoleHandler(writer, config)
	default:
		handler = newConsoleHandler(writer, config)
	}

	// Wrap handler with hooks if any
	if len(config.Hooks) > 0 || len(getGlobalHooks()) > 0 {
		handler = &hookHandler{
			handler: handler,
			hooks:   append(getGlobalHooks(), config.Hooks...),
			level:   config.Level,
		}
	}

	// Create the slog logger
	slogLogger := slog.New(handler)

	// Add default fields if any
	if len(config.DefaultFields) > 0 {
		slogLogger = slogLogger.With(config.DefaultFields...)
	}

	return &Logger{
		slog:   slogLogger,
		level:  config.Level,
		format: config.Format,
		writer: writer,
		fields: config.DefaultFields,
		hooks:  config.Hooks,
		config: config,
	}
}

// Slog returns the underlying *slog.Logger for compatibility.
func (l *Logger) Slog() *slog.Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.slog
}

// WithFields creates a new logger with additional fields.
// This follows the immutable pattern - returns a new instance.
func (l *Logger) WithFields(args ...any) *Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	newFields := make([]any, len(l.fields)+len(args))
	copy(newFields, l.fields)
	copy(newFields[len(l.fields):], args)

	return &Logger{
		slog:   l.slog.With(args...),
		level:  l.level,
		format: l.format,
		writer: l.writer,
		fields: newFields,
		hooks:  l.hooks,
		config: l.config,
	}
}

// IsLevelEnabled returns true if the given level is enabled.
func (l *Logger) IsLevelEnabled(level Level) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return level >= l.level
}

// log is the internal logging function that handles all levels.
func (l *Logger) log(level Level, msg string, args ...any) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if !l.IsLevelEnabled(level) {
		return
	}

	// Get caller information if needed
	var pcs [1]uintptr
	if l.config.ShowCaller {
		runtime.Callers(l.config.CallerSkip+1, pcs[:])
	}

	// Create record
	r := slog.NewRecord(time.Now(), level.ToSlogLevel(), msg, pcs[0])
	r.Add(args...)

	// Handle the record
	_ = l.slog.Handler().Handle(context.Background(), r)

	// If this is a FATAL level, call exit handler
	if level == FATAL {
		if l.config.EnableStackTrace >= FATAL {
			// TODO: Add stack trace
		}
		getExitHandler()(1)
	}
}

// Trace logs at TRACE level.
func (l *Logger) Trace(msg string, args ...any) {
	l.log(TRACE, msg, args...)
}

// Debug logs at DEBUG level.
func (l *Logger) Debug(msg string, args ...any) {
	l.log(DEBUG, msg, args...)
}

// Info logs at INFO level.
func (l *Logger) Info(msg string, args ...any) {
	l.log(INFO, msg, args...)
}

// Warn logs at WARN level.
func (l *Logger) Warn(msg string, args ...any) {
	l.log(WARN, msg, args...)
}

// Error logs at ERROR level.
func (l *Logger) Error(msg string, args ...any) {
	l.log(ERROR, msg, args...)
}

// Fatal logs at FATAL level and then exits.
func (l *Logger) Fatal(msg string, args ...any) {
	l.log(FATAL, msg, args...)
}

// Writer returns an io.Writer that writes to the logger at the given level.
func (l *Logger) Writer(level Level) io.Writer {
	return &logWriter{
		logger: l,
		level:  level,
	}
}

// logWriter is an io.Writer that writes to a Logger at a specific level.
type logWriter struct {
	logger *Logger
	level  Level
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	w.logger.log(w.level, string(p))
	return len(p), nil
}

// multiWriter writes to multiple writers simultaneously.
type multiWriter struct {
	writers []io.Writer
}

func (mw *multiWriter) Write(p []byte) (n int, err error) {
	for _, w := range mw.writers {
		n, err = w.Write(p)
		if err != nil {
			return n, err
		}
	}
	return len(p), nil
}

// hookHandler wraps a handler and calls hooks before logging.
type hookHandler struct {
	handler slog.Handler
	hooks   []Hook
	level   Level
}

func (h *hookHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *hookHandler) Handle(ctx context.Context, r slog.Record) error {
	// Create entry for hooks
	entry := &Entry{
		Level:   Level(r.Level),
		Message: r.Message,
		Time:    r.Time,
		Fields:  []any{},
	}

	// Extract fields
	r.Attrs(func(a slog.Attr) bool {
		entry.Fields = append(entry.Fields, a.Key, a.Value.Any())
		if a.Key == "error" {
			if err, ok := a.Value.Any().(error); ok {
				entry.Error = err
			}
		}
		return true
	})

	// Call hooks
	for _, hook := range h.hooks {
		if err := hook(entry); err != nil {
			return err
		}
	}

	return h.handler.Handle(ctx, r)
}

func (h *hookHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &hookHandler{
		handler: h.handler.WithAttrs(attrs),
		hooks:   h.hooks,
		level:   h.level,
	}
}

func (h *hookHandler) WithGroup(name string) slog.Handler {
	return &hookHandler{
		handler: h.handler.WithGroup(name),
		hooks:   h.hooks,
		level:   h.level,
	}
}
