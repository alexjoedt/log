package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorGray   = "\033[90m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[97m"
)

// consoleHandler is a custom slog.Handler for beautiful console output.
type consoleHandler struct {
	writer    io.Writer
	level     slog.Level
	useColors bool
	config    *Config
	attrs     []slog.Attr
	groups    []string
}

// newConsoleHandler creates a new console handler.
func newConsoleHandler(w io.Writer, config *Config) *consoleHandler {
	return &consoleHandler{
		writer:    w,
		level:     config.Level.ToSlogLevel(),
		useColors: isTerminal(w),
		config:    config,
		attrs:     []slog.Attr{},
		groups:    []string{},
	}
}

// isTerminal detects if the writer is a terminal (TTY).
func isTerminal(w io.Writer) bool {
	// Check if writer is os.Stdout or os.Stderr
	if f, ok := w.(*os.File); ok {
		// Check if it's a terminal using file descriptor
		fd := f.Fd()
		return isatty(int(fd))
	}
	return false
}

// isatty checks if a file descriptor is a terminal.
// This is a simplified version - for production use a library like mattn/go-isatty
func isatty(fd int) bool {
	// Simple check for Unix-like systems
	var termios [128]byte
	_, _, errno := syscallTermios(fd, &termios)
	return errno == 0
}

// Enabled reports whether the handler handles records at the given level.
func (h *consoleHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle formats and writes a log record.
func (h *consoleHandler) Handle(_ context.Context, r slog.Record) error {
	buf := make([]byte, 0, 1024)

	// Timestamp
	if h.config.TimestampFormat != "" {
		timestamp := r.Time.Format(h.config.TimestampFormat)
		if h.useColors {
			buf = append(buf, colorGray...)
		}
		buf = append(buf, timestamp...)
		if h.useColors {
			buf = append(buf, colorReset...)
		}
		buf = append(buf, ' ')
	}

	// Level with color
	level := Level(r.Level)
	buf = append(buf, h.formatLevel(level)...)
	buf = append(buf, ' ')

	// Caller info if enabled
	if h.config.ShowCaller && r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		file := filepath.Base(f.File)
		caller := file + ":" + strconv.Itoa(f.Line)

		if h.useColors {
			buf = append(buf, colorGray...)
		}
		buf = append(buf, caller...)
		if h.useColors {
			buf = append(buf, colorReset...)
		}
		buf = append(buf, ' ')
	}

	// Message
	buf = append(buf, r.Message...)

	// Attributes
	r.Attrs(func(a slog.Attr) bool {
		buf = append(buf, ' ')
		buf = h.appendAttr(buf, a)
		return true
	})

	// Add newline
	buf = append(buf, '\n')

	// Write
	_, err := h.writer.Write(buf)
	return err
}

// formatLevel formats the level with optional colors.
func (h *consoleHandler) formatLevel(level Level) string {
	var color string
	var levelStr string

	switch level {
	case TRACE:
		color = colorCyan
		levelStr = "TRACE"
	case DEBUG:
		color = colorBlue
		levelStr = "DEBUG"
	case INFO:
		color = colorWhite
		levelStr = "INFO "
	case WARN:
		color = colorYellow
		levelStr = "WARN "
	case ERROR:
		color = colorRed
		levelStr = "ERROR"
	case FATAL:
		color = colorRed
		levelStr = "FATAL"
	default:
		color = colorWhite
		levelStr = "UNKNOWN"
	}

	if h.useColors {
		return fmt.Sprintf("%s[%s]%s", color, levelStr, colorReset)
	}
	return fmt.Sprintf("[%s]", levelStr)
}

// appendAttr appends a formatted attribute to the buffer.
func (h *consoleHandler) appendAttr(buf []byte, a slog.Attr) []byte {
	// Handle groups
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		if a.Key != "" {
			buf = append(buf, a.Key...)
			buf = append(buf, '=')
		}
		buf = append(buf, '{')
		for i, attr := range attrs {
			if i > 0 {
				buf = append(buf, ' ')
			}
			buf = h.appendAttr(buf, attr)
		}
		buf = append(buf, '}')
		return buf
	}

	// Key
	if h.useColors {
		buf = append(buf, colorCyan...)
	}
	buf = append(buf, a.Key...)
	if h.useColors {
		buf = append(buf, colorReset...)
	}
	buf = append(buf, '=')

	// Value
	buf = h.appendValue(buf, a.Value)
	return buf
}

// appendValue appends a formatted value to the buffer.
func (h *consoleHandler) appendValue(buf []byte, v slog.Value) []byte {
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

// WithAttrs returns a new handler with additional attributes.
func (h *consoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &consoleHandler{
		writer:    h.writer,
		level:     h.level,
		useColors: h.useColors,
		config:    h.config,
		attrs:     newAttrs,
		groups:    h.groups,
	}
}

// WithGroup returns a new handler with a group name.
func (h *consoleHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name

	return &consoleHandler{
		writer:    h.writer,
		level:     h.level,
		useColors: h.useColors,
		config:    h.config,
		attrs:     h.attrs,
		groups:    newGroups,
	}
}

// syscallTermios is a platform-specific syscall for terminal detection.
// For simplicity, we'll implement a basic version.
func syscallTermios(fd int, termios *[128]byte) (uintptr, uintptr, int) {
	// This is a simplified placeholder
	// In production, use golang.org/x/term or mattn/go-isatty
	if fd == int(os.Stdout.Fd()) || fd == int(os.Stderr.Fd()) {
		return 0, 0, 0
	}
	return 0, 0, 1
}
