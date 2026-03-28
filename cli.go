package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
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
// It renders symbol-prefixed messages. Key=value fields are appended when
// useFields is true (controlled via WithCLIFields).
type cliHandler struct {
	writer     io.Writer
	leveler    slog.Leveler
	useColors  bool
	useSymbols bool
	useFields  bool
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
		useFields:  config.CLIFields,
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
// Format: [symbol] message key=value key=value …
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

	// Message
	buf = append(buf, r.Message...)

	if h.useFields {
		// Persistent attributes (added via WithAttrs)
		for _, a := range h.attrs {
			pre := len(buf)
			buf = append(buf, ' ')
			var wrote bool
			buf, wrote = h.appendAttr(buf, a, h.groups)
			if !wrote {
				buf = buf[:pre]
			}
		}

		// Per-record attributes
		r.Attrs(func(a slog.Attr) bool {
			pre := len(buf)
			buf = append(buf, ' ')
			var wrote bool
			buf, wrote = h.appendAttr(buf, a, h.groups)
			if !wrote {
				buf = buf[:pre]
			}
			return true
		})
	}

	// Add newline
	buf = append(buf, '\n')

	// Write
	_, err := h.writer.Write(buf)
	return err
}

// appendAttr appends a single slog.Attr as key=value to buf.
// groups is the current group prefix stack.
// It returns the updated buffer and whether any content was written.
// Callers must add a separator (space) only when this returns true.
func (h *cliHandler) appendAttr(buf []byte, a slog.Attr, groups []string) ([]byte, bool) {
	a.Value = a.Value.Resolve()

	if a.Value.Kind() == slog.KindGroup {
		group := a.Key
		newGroups := groups
		if group != "" {
			newGroups = append(newGroups, group)
		}
		n := len(buf)
		for _, ga := range a.Value.Group() {
			attrStart := len(buf)
			if len(buf) > n {
				buf = append(buf, ' ')
			}
			var wrote bool
			buf, wrote = h.appendAttr(buf, ga, newGroups)
			if !wrote {
				buf = buf[:attrStart]
			}
		}
		return buf, len(buf) > n
	}

	// Build the full key with group prefix
	key := a.Key
	if len(groups) > 0 {
		key = strings.Join(groups, ".") + "." + key
	}

	if h.useColors {
		buf = append(buf, colorCyan...)
	}
	buf = append(buf, key...)
	if h.useColors {
		buf = append(buf, colorReset...)
	}
	buf = append(buf, '=')
	buf = h.appendValue(buf, a.Value)
	return buf, true
}

// appendValue formats a slog.Value into buf.
func (h *cliHandler) appendValue(buf []byte, v slog.Value) []byte {
	switch v.Kind() {
	case slog.KindString:
		return append(buf, v.String()...)
	case slog.KindInt64:
		return strconv.AppendInt(buf, v.Int64(), 10)
	case slog.KindUint64:
		return strconv.AppendUint(buf, v.Uint64(), 10)
	case slog.KindFloat64:
		return strconv.AppendFloat(buf, v.Float64(), 'f', -1, 64)
	case slog.KindBool:
		return strconv.AppendBool(buf, v.Bool())
	case slog.KindTime:
		return append(buf, v.Time().Format(time.RFC3339)...)
	case slog.KindDuration:
		return append(buf, v.Duration().String()...)
	default:
		return append(buf, fmt.Sprint(v.Any())...)
	}
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

// WithAttrs returns a new handler with additional attributes.
func (h *cliHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &cliHandler{
		writer:     h.writer,
		leveler:    h.leveler,
		useColors:  h.useColors,
		useSymbols: h.useSymbols,
		useFields:  h.useFields,
		config:     h.config,
		attrs:      newAttrs,
		groups:     h.groups,
	}
}

// WithGroup returns a new handler with a group name.
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
		useFields:  h.useFields,
		config:     h.config,
		attrs:      h.attrs,
		groups:     newGroups,
	}
}

// NewCLILogger creates a logger optimized for CLI applications.
// It writes to stderr, disables timestamps, enables symbol prefixes, and
// appends structured key=value fields after the message by default.
//
// Defaults:
//   - Format:  FormatCLI
//   - Writer:  os.Stderr
//   - Level:   INFO
//   - Symbols: enabled (WithCLISymbols)
//   - Fields:  enabled (WithCLIFields(true))
//
// Example:
//
//	logger := log.NewCLILogger()
//	logger.Step("Building application")
//	logger.Info("processed", "count", 42)
//	logger.Success("Build complete")
//	logger.Failure("Push failed")
//
// Options can be provided to override defaults:
//
//	logger := log.NewCLILogger(
//	    log.WithLevel(log.DEBUG),
//	    log.WithCLIFields(false), // suppress key=value fields
//	)
func NewCLILogger(opts ...Option) *Logger {
	// Set CLI-friendly defaults
	defaults := []Option{
		WithFormat(FormatCLI),
		WithWriter(os.Stderr),
		WithoutTimestamp(),
		WithCLISymbols(),
		WithCLIFields(true),
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
