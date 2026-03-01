package log

import (
	"context"
	"io"
	"log/slog"
	"os"
)

// CLI symbols for different log levels
const (
	symbolSuccess = "✓"
	symbolFailure = "✗"
	symbolWarning = "⚠"
	symbolInfo    = "•"
	symbolDebug   = "→"
)

// CLI colors (reuse from console)
const (
	colorGreen = "\033[32m"
)

// cliHandler is a custom slog.Handler optimized for CLI output.
// It renders clean, symbol-prefixed messages without timestamps or structured fields.
type cliHandler struct {
	writer     io.Writer
	leveler    slog.Leveler
	useColors  bool
	useSymbols bool
	config     *Config
	attrs      []slog.Attr
	groups     []string
}

// newCLIHandler creates a new CLI handler.
func newCLIHandler(w io.Writer, config *Config) *cliHandler {
	return &cliHandler{
		writer:     w,
		leveler:    config.leveler,
		useColors:  isTerminal(w),
		useSymbols: config.CLISymbols,
		config:     config,
		attrs:      []slog.Attr{},
		groups:     []string{},
	}
}

// Enabled reports whether the handler handles records at the given level.
func (h *cliHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.leveler.Level()
}

// Handle formats and writes a log record in CLI style.
// Format: [symbol] message
// Fields and structured data are intentionally ignored for clean CLI output.
func (h *cliHandler) Handle(_ context.Context, r slog.Record) error {
	buf := make([]byte, 0, 256)

	// Add symbol prefix (if enabled)
	if h.useSymbols {
		level := Level(r.Level)
		symbol := h.getSymbol(level)

		if h.useColors {
			color := h.getColor(level)
			buf = append(buf, color...)
			buf = append(buf, symbol...)
			buf = append(buf, colorReset...)
		} else {
			buf = append(buf, symbol...)
		}
		buf = append(buf, ' ')
	}

	// Message only - no fields rendered
	buf = append(buf, r.Message...)

	// Add newline
	buf = append(buf, '\n')

	// Write
	_, err := h.writer.Write(buf)
	return err
}

// getSymbol returns the appropriate symbol for a log level.
func (h *cliHandler) getSymbol(level Level) string {
	switch level {
	case TRACE:
		return symbolDebug
	case DEBUG:
		return symbolDebug
	case INFO:
		return symbolInfo
	case WARN:
		return symbolWarning
	case ERROR:
		return symbolFailure
	case FATAL:
		return symbolFailure
	default:
		return symbolInfo
	}
}

// getColor returns the appropriate color for a log level.
func (h *cliHandler) getColor(level Level) string {
	switch level {
	case TRACE:
		return colorGray
	case DEBUG:
		return colorBlue
	case INFO:
		return colorWhite
	case WARN:
		return colorYellow
	case ERROR:
		return colorRed
	case FATAL:
		return colorRed
	default:
		return colorWhite
	}
}

// getSuccessColor returns green for success messages.
func (h *cliHandler) getSuccessColor() string {
	return colorGreen
}

// WithAttrs returns a new handler with additional attributes.
// Note: CLI handler ignores attributes for clean output.
func (h *cliHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &cliHandler{
		writer:     h.writer,
		leveler:    h.leveler,
		useColors:  h.useColors,
		useSymbols: h.useSymbols,
		config:     h.config,
		attrs:      newAttrs,
		groups:     h.groups,
	}
}

// WithGroup returns a new handler with a group name.
// Note: CLI handler ignores groups for clean output.
func (h *cliHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name

	return &cliHandler{
		writer:     h.writer,
		leveler:    h.leveler,
		useColors:  h.useColors,
		useSymbols: h.useSymbols,
		config:     h.config,
		attrs:      h.attrs,
		groups:     newGroups,
	}
}

// NewCLILogger creates a logger optimized for CLI applications.
// It uses stderr by default, disables timestamps, and renders clean
// symbol-prefixed messages without structured fields.
//
// Example:
//
//	logger := log.NewCLILogger()
//	logger.Step("Building application")
//	logger.Success("Build complete")
//	logger.Failure("Push failed")
//
// Options can be provided to customize behavior:
//
//	logger := log.NewCLILogger(
//	    log.WithLevel(log.DEBUG),
//	    log.WithCLISymbols(false), // disable symbols
//	)
func NewCLILogger(opts ...Option) *Logger {
	// Set CLI-friendly defaults
	defaults := []Option{
		WithFormat(FormatCLI),
		WithWriter(os.Stderr),
		WithoutTimestamp(),
		WithCLISymbols(),
		WithLevel(INFO),
	}

	// User options override defaults
	return New(append(defaults, opts...)...)
}

// handleCLISuccess handles success messages specially for CLI output.
func (h *cliHandler) handleCLISuccess(msg string) error {
	buf := make([]byte, 0, 256)

	// Add success symbol
	if h.useSymbols {
		if h.useColors {
			buf = append(buf, colorGreen...)
			buf = append(buf, symbolSuccess...)
			buf = append(buf, colorReset...)
		} else {
			buf = append(buf, symbolSuccess...)
		}
		buf = append(buf, ' ')
	}

	// Message
	buf = append(buf, msg...)
	buf = append(buf, '\n')

	// Write
	_, err := h.writer.Write(buf)
	return err
}

// handleCLIFailure handles failure messages specially for CLI output.
func (h *cliHandler) handleCLIFailure(msg string) error {
	buf := make([]byte, 0, 256)

	// Add failure symbol
	if h.useSymbols {
		if h.useColors {
			buf = append(buf, colorRed...)
			buf = append(buf, symbolFailure...)
			buf = append(buf, colorReset...)
		} else {
			buf = append(buf, symbolFailure...)
		}
		buf = append(buf, ' ')
	}

	// Message
	buf = append(buf, msg...)
	buf = append(buf, '\n')

	// Write
	_, err := h.writer.Write(buf)
	return err
}

// handleCLIStep handles step messages specially for CLI output.
func (h *cliHandler) handleCLIStep(msg string) error {
	buf := make([]byte, 0, 256)

	// Add info symbol
	if h.useSymbols {
		if h.useColors {
			buf = append(buf, colorBlue...)
			buf = append(buf, symbolInfo...)
			buf = append(buf, colorReset...)
		} else {
			buf = append(buf, symbolInfo...)
		}
		buf = append(buf, ' ')
	}

	// Message
	buf = append(buf, msg...)
	buf = append(buf, '\n')

	// Write
	_, err := h.writer.Write(buf)
	return err
}
