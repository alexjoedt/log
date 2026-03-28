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

// removeTimeAttr is a ReplaceAttr function that removes the time attribute from log records.
func removeTimeAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey && len(groups) == 0 {
		return slog.Attr{}
	}
	return a
}

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
	leveler          slog.Leveler // Minimum log level (supports dynamic levels via slog.LevelVar)
	Format           Format
	Writer           io.Writer
	TimestampFormat  string
	DisableTimestamp bool
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

	// CLI settings
	CLISymbols bool // Enable symbol prefixes for CLI format
	CLIFields  bool // Render key=value fields in CLI format; opt-in via WithCLIFields(true) or NewCLILogger
}

// Level returns the current minimum log level.
// This method allows checking the current level even when using dynamic levels.
func (c *Config) Level() Level {
	if c.leveler == nil {
		return INFO
	}
	return Level(c.leveler.Level())
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

// Option is a function that configures a Logger.
type Option func(*Config)

// WithLevel sets the minimum log level.
// Accepts slog.Leveler interface, supporting:
//   - Static levels: log.DEBUG, log.INFO, etc.
//   - slog levels: slog.LevelDebug, slog.LevelInfo, etc.
//   - Dynamic levels: &slog.LevelVar{} (can be changed at runtime)
//
// Example with static level:
//
//	logger := log.New(log.WithLevel(log.DEBUG))
//
// Example with dynamic level:
//
//	logLevel := &slog.LevelVar{}
//	logLevel.Set(slog.LevelInfo)
//	logger := log.New(log.WithLevel(logLevel))
//	// Later: logLevel.Set(slog.LevelDebug)
func WithLevel(leveler slog.Leveler) Option {
	return func(c *Config) {
		if leveler != nil {
			c.leveler = leveler
		}
	}
}

// WithFormat sets the output format.
func WithFormat(format Format) Option {
	return func(c *Config) {
		c.Format = format
	}
}

// WithWriter sets the output writer.
func WithWriter(w io.Writer) Option {
	return func(c *Config) {
		c.Writer = w
	}
}

// WithWriters sets multiple output writers (fan-out).
func WithWriters(writers ...io.Writer) Option {
	return func(c *Config) {
		c.Writers = writers
	}
}

// WithTimestampFormat sets the timestamp format.
func WithTimestampFormat(format string) Option {
	return func(c *Config) {
		c.TimestampFormat = format
	}
}

// WithoutTimestamp disables timestamp output in logs.
func WithoutTimestamp() Option {
	return func(c *Config) {
		c.DisableTimestamp = true
	}
}

// WithCaller enables caller information.
func WithCaller() Option {
	return func(c *Config) {
		c.ShowCaller = true
	}
}

// WithCallerSkip sets the number of stack frames to skip for caller info.
func WithCallerSkip(skip int) Option {
	return func(c *Config) {
		c.CallerSkip = skip
	}
}

// WithDefaultFields adds default fields to all log messages.
func WithDefaultFields(args ...any) Option {
	return func(c *Config) {
		c.DefaultFields = append(c.DefaultFields, args...)
	}
}

// WithStackTrace enables stack traces for the given level and above.
func WithStackTrace(level Level) Option {
	return func(c *Config) {
		c.EnableStackTrace = level
	}
}

// WithRotation configures log rotation.
func WithRotation(maxSize, maxBackups, maxAge int, compress bool) Option {
	return func(c *Config) {
		c.Rotation = &RotationConfig{
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     maxAge,
			Compress:   compress,
		}
	}
}

// WithSampling configures log sampling.
func WithSampling(first, thereafter int) Option {
	return func(c *Config) {
		c.Sampling = &SamplingConfig{
			First:      first,
			Thereafter: thereafter,
		}
	}
}

// WithBuffer configures buffered writing.
func WithBuffer(size int, flushInterval time.Duration) Option {
	return func(c *Config) {
		c.Buffer = &BufferConfig{
			Size:          size,
			FlushInterval: flushInterval,
		}
	}
}

// WithHook adds a hook to this logger.
func WithHook(hook Hook) Option {
	return func(c *Config) {
		c.Hooks = append(c.Hooks, hook)
	}
}

// buildHandler constructs a fully-configured slog.Handler from Config.
// It applies writer wrappers (rotation, sampling, buffering), creates the
// appropriate handler based on Format, and applies hook wrappers if configured.
func buildHandler(config *Config) slog.Handler {
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
		opts := &slog.HandlerOptions{
			Level:     config.leveler,
			AddSource: config.ShowCaller,
		}
		if config.DisableTimestamp {
			opts.ReplaceAttr = removeTimeAttr
		}
		handler = slog.NewJSONHandler(writer, opts)
	case FormatText:
		opts := &slog.HandlerOptions{
			Level:     config.leveler,
			AddSource: config.ShowCaller,
		}
		if config.DisableTimestamp {
			opts.ReplaceAttr = removeTimeAttr
		}
		handler = slog.NewTextHandler(writer, opts)
	case FormatConsole:
		handler = newConsoleHandler(writer, config)
	case FormatCLI:
		handler = newCLIHandler(writer, config)
	default:
		handler = newConsoleHandler(writer, config)
	}

	// Wrap handler with hooks if any
	if len(config.Hooks) > 0 || len(getGlobalHooks()) > 0 {
		handler = &hookHandler{
			handler: handler,
			hooks:   append(getGlobalHooks(), config.Hooks...),
			leveler: config.leveler,
		}
	}

	return handler
}

// New creates a new Logger with the given options.
func New(opts ...Option) *Logger {
	// Create default config
	config := &Config{
		leveler:         getDefaultLevel(),
		Format:          getDefaultFormat(),
		Writer:          os.Stderr,
		TimestampFormat: time.RFC3339,
		CallerSkip:      2,
		DefaultFields:   []any{},
		Hooks:           []Hook{},
		CLIFields:       false,
	}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	// Build the handler
	handler := buildHandler(config)

	// Determine the writer for Logger struct
	var writer io.Writer
	if len(config.Writers) > 1 {
		writer = &multiWriter{writers: config.Writers}
	} else if len(config.Writers) == 1 {
		writer = config.Writers[0]
	} else {
		writer = config.Writer
	}

	// Create the slog logger
	slogLogger := slog.New(handler)

	// Add default fields if any
	if len(config.DefaultFields) > 0 {
		slogLogger = slogLogger.With(config.DefaultFields...)
	}

	return &Logger{
		slog:   slogLogger,
		level:  config.Level(),
		format: config.Format,
		writer: writer,
		fields: config.DefaultFields,
		hooks:  config.Hooks,
		config: config,
	}
}

// NewSlogHandler creates a fully-configured slog.Handler with the given options.
// This allows using the package's enhanced features (rotation, sampling, console
// formatting, hooks) while maintaining a pure slog-based workflow.
//
// All options that work with New() also work with NewSlogHandler(), including:
//   - WithLevel, WithFormat, WithWriter
//   - WithRotation, WithSampling, WithBuffer
//   - WithHook, WithCaller, WithoutTimestamp
//   - WithDefaultFields (applied via handler.WithAttrs)
//
// Example usage:
//
//	handler := log.NewSlogHandler(
//	    log.WithLevel(log.DEBUG),
//	    log.WithFormat(log.FormatJSON),
//	    log.WithDefaultFields("service", "api", "version", "1.0"),
//	)
//	logger := slog.New(handler)
//	logger.Info("using slog with enhanced handler")
//
// Note: When using buffered or rotating writers, the caller is responsible for
// managing writer lifecycle (flushing/closing) if needed.
func NewSlogHandler(opts ...Option) slog.Handler {
	// Create default config
	config := &Config{
		leveler:         getDefaultLevel(),
		Format:          getDefaultFormat(),
		Writer:          os.Stderr,
		TimestampFormat: time.RFC3339,
		CallerSkip:      2,
		DefaultFields:   []any{},
		Hooks:           []Hook{},
		CLIFields:       false,
	}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	// Build the handler
	handler := buildHandler(config)

	// Apply default fields to handler if configured
	if len(config.DefaultFields) > 0 {
		var attrs []slog.Attr
		for i := 0; i < len(config.DefaultFields); i += 2 {
			if i+1 < len(config.DefaultFields) {
				key, ok := config.DefaultFields[i].(string)
				if !ok {
					// Convert non-string keys to string
					key = slog.AnyValue(config.DefaultFields[i]).String()
				}
				value := config.DefaultFields[i+1]
				attrs = append(attrs, slog.Any(key, value))
			}
		}
		if len(attrs) > 0 {
			handler = handler.WithAttrs(attrs)
		}
	}

	return handler
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
// This checks against the current log level, which may be dynamic.
func (l *Logger) IsLevelEnabled(level Level) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return level.Level() >= l.config.leveler.Level()
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

// Success logs a success message (CLI-friendly, message only).
// This is intended for CLI applications to indicate successful operations.
// When using FormatCLI, it renders with a green checkmark symbol.
// For other formats, it logs at INFO level.
//
// Example:
//
//	logger.Success("Deployment completed successfully")
func (l *Logger) Success(msg string) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// For CLI format, write directly to handler for custom rendering
	if l.format == FormatCLI {
		if cliHandler, ok := l.slog.Handler().(*cliHandler); ok {
			_ = cliHandler.handleCLISuccess(msg)
			return
		}
		// If handler is wrapped, fall through to regular logging
	}

	// For other formats, just log at INFO level
	l.log(INFO, msg)
}

// Failure logs a failure message (CLI-friendly, message only).
// This is intended for CLI applications to indicate failed operations.
// When using FormatCLI, it renders with a red X symbol.
// For other formats, it logs at ERROR level.
//
// Example:
//
//	logger.Failure("Deployment failed")
func (l *Logger) Failure(msg string) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// For CLI format, write directly to handler for custom rendering
	if l.format == FormatCLI {
		if cliHandler, ok := l.slog.Handler().(*cliHandler); ok {
			_ = cliHandler.handleCLIFailure(msg)
			return
		}
		// If handler is wrapped, fall through to regular logging
	}

	// For other formats, just log at ERROR level
	l.log(ERROR, msg)
}

// Step logs a step/progress message (CLI-friendly, message only).
// This is intended for CLI applications to indicate progress or steps.
// When using FormatCLI, it renders with a blue bullet symbol.
// For other formats, it logs at INFO level.
//
// Example:
//
//	logger.Step("Building application...")
func (l *Logger) Step(msg string) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// For CLI format, write directly to handler for custom rendering
	if l.format == FormatCLI {
		if cliHandler, ok := l.slog.Handler().(*cliHandler); ok {
			_ = cliHandler.handleCLIStep(msg)
			return
		}
		// If handler is wrapped, fall through to regular logging
	}

	// For other formats, just log at INFO level
	l.log(INFO, msg)
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
	leveler slog.Leveler
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
		leveler: h.leveler,
	}
}

func (h *hookHandler) WithGroup(name string) slog.Handler {
	return &hookHandler{
		handler: h.handler.WithGroup(name),
		hooks:   h.hooks,
		leveler: h.leveler,
	}
}
