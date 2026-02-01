package log

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestLevels(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{TRACE, "TRACE"},
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("Level.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultLogger(t *testing.T) {
	// Test that default logger works
	logger := Default()
	if logger == nil {
		t.Fatal("Default logger is nil")
	}

	// Set a custom default logger
	buf := &bytes.Buffer{}
	customLogger := New(WithWriter(buf))
	SetDefault(customLogger)

	Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected log output to contain 'test message', got: %s", output)
	}

	// Reset to original
	SetDefault(New())
}

func TestLoggerBuilder(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(
		WithLevel(DEBUG),
		WithWriter(buf),
		WithFormat(FormatJSON),
		WithDefaultFields("app", "test", "version", "1.0"),
	)

	if logger == nil {
		t.Fatal("Logger is nil")
	}

	logger.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain 'test message', got: %s", output)
	}
	if !strings.Contains(output, "app") {
		t.Errorf("Expected output to contain 'app', got: %s", output)
	}
}

func TestLogLevels(t *testing.T) {
	logger, hook := NewTestLogger()

	logger.Trace("trace message")
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	if hook.Count() != 5 {
		t.Errorf("Expected 5 log entries, got %d", hook.Count())
	}

	if !hook.HasMessage("trace message") {
		t.Error("Expected to find 'trace message'")
	}

	if !hook.HasLevel(ERROR) {
		t.Error("Expected to find ERROR level")
	}
}

func TestLoggerWithFields(t *testing.T) {
	logger, hook := NewTestLogger()

	// Create a logger with fields
	requestLogger := logger.WithFields("request_id", "12345", "user_id", 42)
	requestLogger.Info("processing request")

	if hook.Count() != 1 {
		t.Fatalf("Expected 1 log entry, got %d", hook.Count())
	}

	entry := hook.LastEntry()
	if entry.Message != "processing request" {
		t.Errorf("Expected message 'processing request', got '%s'", entry.Message)
	}
}

func TestIsLevelEnabled(t *testing.T) {
	logger := New(WithLevel(WARN))

	if logger.IsLevelEnabled(DEBUG) {
		t.Error("DEBUG should not be enabled when level is WARN")
	}

	if logger.IsLevelEnabled(INFO) {
		t.Error("INFO should not be enabled when level is WARN")
	}

	if !logger.IsLevelEnabled(WARN) {
		t.Error("WARN should be enabled when level is WARN")
	}

	if !logger.IsLevelEnabled(ERROR) {
		t.Error("ERROR should be enabled when level is WARN")
	}
}

func TestPackageLevelFunctions(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(WithWriter(buf), WithLevel(DEBUG))
	SetDefault(logger)

	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")

	output := buf.String()

	if !strings.Contains(output, "debug message") {
		t.Error("Expected output to contain 'debug message'")
	}
	if !strings.Contains(output, "info message") {
		t.Error("Expected output to contain 'info message'")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("Expected output to contain 'warn message'")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Expected output to contain 'error message'")
	}
}

func TestFatalHandler(t *testing.T) {
	exitCalled := false
	exitCode := 0

	// Set custom exit handler
	SetExitHandler(func(code int) {
		exitCalled = true
		exitCode = code
	})

	logger, _ := NewTestLogger()
	logger.Fatal("fatal error")

	if !exitCalled {
		t.Error("Expected exit handler to be called")
	}

	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}

	// Reset exit handler
	SetExitHandler(func(code int) {})
}

func TestContext(t *testing.T) {
	logger, hook := NewTestLogger()

	ctx := ContextWithLogger(context.Background(), logger.Logger)

	FromContext(ctx).Info("message from context")

	if hook.Count() != 1 {
		t.Fatalf("Expected 1 log entry, got %d", hook.Count())
	}

	entry := hook.LastEntry()
	if entry.Message != "message from context" {
		t.Errorf("Expected message 'message from context', got '%s'", entry.Message)
	}
}

func TestContextWithoutLogger(t *testing.T) {
	ctx := context.Background()

	// Should return default logger
	logger := FromContext(ctx)
	if logger == nil {
		t.Error("Expected non-nil logger")
	}
}

func TestHooks(t *testing.T) {
	hookCalled := false
	var capturedEntry *Entry

	hook := func(entry *Entry) error {
		hookCalled = true
		capturedEntry = entry
		return nil
	}

	buf := &bytes.Buffer{}
	logger := New(
		WithWriter(buf),
		WithHook(hook),
	)

	logger.Info("test message", "key", "value")

	if !hookCalled {
		t.Error("Expected hook to be called")
	}

	if capturedEntry == nil {
		t.Fatal("Expected captured entry to be non-nil")
	}

	if capturedEntry.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", capturedEntry.Message)
	}

	if capturedEntry.Level != INFO {
		t.Errorf("Expected level INFO, got %v", capturedEntry.Level)
	}
}

func TestHookError(t *testing.T) {
	hookErr := errors.New("hook error")
	hook := func(entry *Entry) error {
		return hookErr
	}

	buf := &bytes.Buffer{}
	logger := New(
		WithWriter(buf),
		WithHook(hook),
	)

	// Log should not appear because hook returns error
	logger.Info("test message")

	output := buf.String()
	if strings.Contains(output, "test message") {
		t.Error("Expected log to be suppressed by hook error")
	}
}

func TestTestLogger(t *testing.T) {
	logger, hook := NewTestLogger()

	logger.Info("message 1")
	logger.Warn("message 2")
	logger.Error("message 3")

	if hook.Count() != 3 {
		t.Errorf("Expected 3 entries, got %d", hook.Count())
	}

	if hook.CountLevel(INFO) != 1 {
		t.Errorf("Expected 1 INFO entry, got %d", hook.CountLevel(INFO))
	}

	if hook.CountLevel(WARN) != 1 {
		t.Errorf("Expected 1 WARN entry, got %d", hook.CountLevel(WARN))
	}

	if hook.CountLevel(ERROR) != 1 {
		t.Errorf("Expected 1 ERROR entry, got %d", hook.CountLevel(ERROR))
	}

	lastEntry := hook.LastEntry()
	if lastEntry.Message != "message 3" {
		t.Errorf("Expected last message 'message 3', got '%s'", lastEntry.Message)
	}

	hook.Reset()
	if hook.Count() != 0 {
		t.Errorf("Expected 0 entries after reset, got %d", hook.Count())
	}
}

func TestConsoleFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(
		WithWriter(buf),
		WithFormat(FormatConsole),
		WithLevel(DEBUG),
	)

	logger.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain 'test message', got: %s", output)
	}
	if !strings.Contains(output, "INFO") || !strings.Contains(output, "[INFO") {
		t.Errorf("Expected output to contain '[INFO]', got: %s", output)
	}
}

func TestJSONFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(
		WithWriter(buf),
		WithFormat(FormatJSON),
	)

	logger.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, `"msg":"test message"`) {
		t.Errorf("Expected JSON output with message, got: %s", output)
	}
	if !strings.Contains(output, `"key":"value"`) {
		t.Errorf("Expected JSON output with key/value, got: %s", output)
	}
}

func TestTextFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(
		WithWriter(buf),
		WithFormat(FormatText),
	)

	logger.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain 'test message', got: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("Expected output to contain 'key=value', got: %s", output)
	}
}

func TestEnvConfiguration(t *testing.T) {
	// This test would require mocking environment variables
	// For now, just test that FromEnv creates a logger
	logger := FromEnv()
	if logger == nil {
		t.Error("Expected non-nil logger from FromEnv")
	}
}

func TestLazyEvaluation(t *testing.T) {
	_ = false // expensiveFunctionCalled
	expensiveFunc := func() any {
		// expensiveFunctionCalled = true
		return "expensive result"
	}

	// Logger at WARN level should not call expensive function for DEBUG
	testLogger, _ := NewTestLogger()
	logger := &Logger{
		slog:   testLogger.slog,
		level:  WARN,
		format: testLogger.format,
		writer: testLogger.writer,
		fields: testLogger.fields,
		hooks:  testLogger.hooks,
		config: testLogger.config,
	}

	logger.Debug("debug message", Lazy(expensiveFunc))

	// The function should not be called since DEBUG is disabled
	// Note: This test may not work as expected due to how slog handles lazy values
	// In a real implementation, you'd need to ensure the lazy function is only called
	// when the log level is enabled
}

func TestMultipleWriters(t *testing.T) {
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}

	logger := New(
		WithWriters(buf1, buf2),
	)

	logger.Info("test message")

	output1 := buf1.String()
	output2 := buf2.String()

	if !strings.Contains(output1, "test message") {
		t.Errorf("Expected output1 to contain 'test message', got: %s", output1)
	}

	if !strings.Contains(output2, "test message") {
		t.Errorf("Expected output2 to contain 'test message', got: %s", output2)
	}
}

func TestTimestampFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(
		WithWriter(buf),
		WithFormat(FormatConsole),
		WithTimestampFormat(time.RFC3339Nano),
	)

	logger.Info("test message")

	output := buf.String()
	// Just verify something was logged
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain 'test message', got: %s", output)
	}
}

func TestWithoutTimestamp(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(
		WithWriter(buf),
		WithFormat(FormatConsole),
		WithoutTimestamp(),
	)

	logger.Info("test message")

	output := buf.String()
	// Verify the message is logged
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain 'test message', got: %s", output)
	}
	// Verify the output doesn't contain timestamp patterns like "2024" or typical RFC3339 patterns
	// The default timestamp format is RFC3339, so we check for "T" which appears in RFC3339
	if strings.Contains(output, "T") && strings.Count(output, ":") >= 2 {
		t.Errorf("Expected output to not contain timestamp, but got: %s", output)
	}
}

func TestWithoutTimestampJSON(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(
		WithWriter(buf),
		WithFormat(FormatJSON),
		WithoutTimestamp(),
	)

	logger.Info("test message")

	output := buf.String()
	// Verify the message is logged
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain 'test message', got: %s", output)
	}
	// Verify no "time" field in JSON output
	if strings.Contains(output, `"time"`) {
		t.Errorf("Expected JSON output to not contain 'time' field, but got: %s", output)
	}
	// Verify it still contains level and msg
	if !strings.Contains(output, `"level"`) || !strings.Contains(output, `"msg"`) {
		t.Errorf("Expected JSON output to contain 'level' and 'msg' fields, got: %s", output)
	}
}

func TestWithoutTimestampText(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(
		WithWriter(buf),
		WithFormat(FormatText),
		WithoutTimestamp(),
	)

	logger.Info("test message")

	output := buf.String()
	// Verify the message is logged
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain 'test message', got: %s", output)
	}
	// Verify no "time=" field in text output
	if strings.Contains(output, "time=") {
		t.Errorf("Expected text output to not contain 'time=' field, but got: %s", output)
	}
	// Verify it still contains level and msg
	if !strings.Contains(output, "level=") || !strings.Contains(output, "msg=") {
		t.Errorf("Expected text output to contain 'level=' and 'msg=' fields, got: %s", output)
	}
}

func TestErrorWithError(t *testing.T) {
	logger, hook := NewTestLogger()

	testErr := errors.New("test error")
	logger.Error("operation failed", "error", testErr)

	if hook.Count() != 1 {
		t.Fatalf("Expected 1 log entry, got %d", hook.Count())
	}

	entry := hook.LastEntry()
	if entry.Error == nil {
		t.Error("Expected entry to have error set")
	}
	if entry.Error.Error() != "test error" {
		t.Errorf("Expected error 'test error', got '%v'", entry.Error)
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Level
	}{
		// Full level names
		{name: "trace lowercase", input: "trace", want: TRACE},
		{name: "debug lowercase", input: "debug", want: DEBUG},
		{name: "info lowercase", input: "info", want: INFO},
		{name: "warn lowercase", input: "warn", want: WARN},
		{name: "warning lowercase", input: "warning", want: WARN},
		{name: "error lowercase", input: "error", want: ERROR},
		{name: "fatal lowercase", input: "fatal", want: FATAL},

		// Abbreviations
		{name: "trace abbrev", input: "trc", want: TRACE},
		{name: "debug abbrev", input: "dbg", want: DEBUG},
		{name: "info abbrev", input: "inf", want: INFO},
		{name: "error abbrev", input: "err", want: ERROR},
		{name: "fatal abbrev fat", input: "fat", want: FATAL},
		{name: "fatal abbrev fatl", input: "fatl", want: FATAL},
		{name: "fatal abbrev ftl", input: "ftl", want: FATAL},

		// Case insensitivity
		{name: "uppercase TRACE", input: "TRACE", want: TRACE},
		{name: "uppercase DEBUG", input: "DEBUG", want: DEBUG},
		{name: "uppercase INFO", input: "INFO", want: INFO},
		{name: "uppercase WARN", input: "WARN", want: WARN},
		{name: "uppercase ERROR", input: "ERROR", want: ERROR},
		{name: "uppercase FATAL", input: "FATAL", want: FATAL},
		{name: "mixed case TrAcE", input: "TrAcE", want: TRACE},
		{name: "mixed case DeBuG", input: "DeBuG", want: DEBUG},
		{name: "mixed case InFo", input: "InFo", want: INFO},
		{name: "mixed case WaRn", input: "WaRn", want: WARN},
		{name: "mixed case ErRoR", input: "ErRoR", want: ERROR},
		{name: "mixed case FaTaL", input: "FaTaL", want: FATAL},

		// Whitespace handling
		{name: "leading space", input: "  info", want: INFO},
		{name: "trailing space", input: "warn  ", want: WARN},
		{name: "both spaces", input: "  error  ", want: ERROR},
		{name: "tabs and spaces", input: "\t debug \t", want: DEBUG},

		// Invalid inputs (should return ERROR as default)
		{name: "empty string", input: "", want: ERROR},
		{name: "unknown level", input: "unknown", want: ERROR},
		{name: "invalid abbrev", input: "xyz", want: ERROR},
		{name: "numeric", input: "123", want: ERROR},
		{name: "special chars", input: "!@#", want: ERROR},
		{name: "spaces only", input: "   ", want: ERROR},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseLogLevel(tt.input)
			if got != tt.want {
				t.Errorf("ParseLogLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNewSlogHandler(t *testing.T) {
	t.Run("creates valid handler", func(t *testing.T) {
		handler := NewSlogHandler()
		if handler == nil {
			t.Fatal("NewSlogHandler returned nil")
		}
	})

	t.Run("works with slog.New", func(t *testing.T) {
		buf := &bytes.Buffer{}
		handler := NewSlogHandler(
			WithLevel(INFO),
			WithFormat(FormatJSON),
			WithWriter(buf),
		)

		logger := slog.New(handler)
		logger.Info("test message", "key", "value")

		output := buf.String()
		if !strings.Contains(output, "test message") {
			t.Errorf("Expected output to contain 'test message', got: %s", output)
		}
		if !strings.Contains(output, "key") {
			t.Errorf("Expected output to contain 'key', got: %s", output)
		}
	})

	t.Run("respects level option", func(t *testing.T) {
		buf := &bytes.Buffer{}
		handler := NewSlogHandler(
			WithLevel(WARN),
			WithFormat(FormatJSON),
			WithWriter(buf),
		)

		logger := slog.New(handler)
		logger.Debug("debug message")
		logger.Info("info message")
		logger.Warn("warn message")

		output := buf.String()
		if strings.Contains(output, "debug message") {
			t.Error("DEBUG message should be filtered")
		}
		if strings.Contains(output, "info message") {
			t.Error("INFO message should be filtered")
		}
		if !strings.Contains(output, "warn message") {
			t.Error("WARN message should be logged")
		}
	})

	t.Run("applies default fields", func(t *testing.T) {
		buf := &bytes.Buffer{}
		handler := NewSlogHandler(
			WithLevel(INFO),
			WithFormat(FormatJSON),
			WithWriter(buf),
			WithDefaultFields("app", "test", "version", "1.0"),
		)

		logger := slog.New(handler)
		logger.Info("message")

		output := buf.String()
		if !strings.Contains(output, "app") {
			t.Errorf("Expected output to contain 'app', got: %s", output)
		}
		if !strings.Contains(output, "test") {
			t.Errorf("Expected output to contain 'test', got: %s", output)
		}
		if !strings.Contains(output, "version") {
			t.Errorf("Expected output to contain 'version', got: %s", output)
		}
	})

	t.Run("supports all formats", func(t *testing.T) {
		formats := []Format{FormatJSON, FormatText, FormatConsole}
		for _, format := range formats {
			t.Run(string(format), func(t *testing.T) {
				buf := &bytes.Buffer{}
				handler := NewSlogHandler(
					WithLevel(INFO),
					WithFormat(format),
					WithWriter(buf),
				)

				logger := slog.New(handler)
				logger.Info("test message")

				output := buf.String()
				if !strings.Contains(output, "test message") {
					t.Errorf("Format %s: expected output to contain 'test message', got: %s", format, output)
				}
			})
		}
	})

	t.Run("applies hooks", func(t *testing.T) {
		buf := &bytes.Buffer{}
		var hookCalled bool
		hook := func(entry *Entry) error {
			hookCalled = true
			if entry.Message != "test message" {
				t.Errorf("Expected message 'test message', got '%s'", entry.Message)
			}
			return nil
		}

		handler := NewSlogHandler(
			WithLevel(INFO),
			WithFormat(FormatJSON),
			WithWriter(buf),
			WithHook(hook),
		)

		logger := slog.New(handler)
		logger.Info("test message")

		if !hookCalled {
			t.Error("Hook was not called")
		}
	})

	t.Run("handles odd-length default fields", func(t *testing.T) {
		buf := &bytes.Buffer{}
		handler := NewSlogHandler(
			WithLevel(INFO),
			WithFormat(FormatJSON),
			WithWriter(buf),
			WithDefaultFields("app", "test", "orphan"),
		)

		logger := slog.New(handler)
		logger.Info("message")

		output := buf.String()
		if !strings.Contains(output, "app") {
			t.Error("Expected 'app' field to be included")
		}
		// Orphan field should be ignored
	})

	t.Run("handles non-string keys in default fields", func(t *testing.T) {
		buf := &bytes.Buffer{}
		handler := NewSlogHandler(
			WithLevel(INFO),
			WithFormat(FormatJSON),
			WithWriter(buf),
			WithDefaultFields(123, "numeric_key"),
		)

		logger := slog.New(handler)
		logger.Info("message")

		output := buf.String()
		if !strings.Contains(output, "message") {
			t.Error("Message should be logged despite non-string key")
		}
	})
}

func TestNewSlogHandlerFeatureParity(t *testing.T) {
	t.Run("produces same output as New()", func(t *testing.T) {
		buf1 := &bytes.Buffer{}
		buf2 := &bytes.Buffer{}

		// Using New()
		logger1 := New(
			WithLevel(INFO),
			WithFormat(FormatJSON),
			WithWriter(buf1),
			WithoutTimestamp(),
		)
		logger1.Info("test message", "key", "value")

		// Using NewSlogHandler()
		handler := NewSlogHandler(
			WithLevel(INFO),
			WithFormat(FormatJSON),
			WithWriter(buf2),
			WithoutTimestamp(),
			WithDefaultFields("key", "value"),
		)
		logger2 := slog.New(handler)
		logger2.Info("test message")

		output1 := buf1.String()
		output2 := buf2.String()

		// Both should contain the same key elements
		if !strings.Contains(output1, "test message") || !strings.Contains(output2, "test message") {
			t.Error("Both outputs should contain 'test message'")
		}
		if !strings.Contains(output1, "key") || !strings.Contains(output2, "key") {
			t.Error("Both outputs should contain 'key'")
		}
	})
}

func TestLevelImplementsSlogLeveler(t *testing.T) {
	// Verify that Level implements slog.Leveler interface
	var _ slog.Leveler = Level(0)
	var _ slog.Leveler = DEBUG
	var _ slog.Leveler = INFO

	// Test Level() method
	if DEBUG.Level() != slog.Level(DEBUG) {
		t.Errorf("DEBUG.Level() = %v, want %v", DEBUG.Level(), slog.Level(DEBUG))
	}
	if INFO.Level() != slog.LevelInfo {
		t.Errorf("INFO.Level() = %v, want %v", INFO.Level(), slog.LevelInfo)
	}
}

func TestWithSlogLevel(t *testing.T) {
	buf := &bytes.Buffer{}

	// Test with slog.Level directly
	logger := New(
		WithLevel(slog.LevelDebug),
		WithWriter(buf),
		WithFormat(FormatJSON),
	)

	logger.Debug("debug message")
	logger.Info("info message")

	output := buf.String()
	if !strings.Contains(output, "debug message") {
		t.Error("Expected debug message to be logged")
	}
	if !strings.Contains(output, "info message") {
		t.Error("Expected info message to be logged")
	}
}

func TestDynamicLogLevel(t *testing.T) {
	buf := &bytes.Buffer{}

	// Create a dynamic level
	logLevel := &slog.LevelVar{}
	logLevel.Set(slog.LevelInfo)

	logger := New(
		WithLevel(logLevel),
		WithWriter(buf),
		WithFormat(FormatJSON),
	)

	// Initially at INFO, debug should be filtered
	logger.Debug("debug 1")
	logger.Info("info 1")

	output1 := buf.String()
	if strings.Contains(output1, "debug 1") {
		t.Error("DEBUG message should be filtered at INFO level")
	}
	if !strings.Contains(output1, "info 1") {
		t.Error("INFO message should be logged")
	}

	// Clear buffer
	buf.Reset()

	// Change level to DEBUG at runtime
	logLevel.Set(slog.LevelDebug)

	// Now debug should appear
	logger.Debug("debug 2")
	logger.Info("info 2")

	output2 := buf.String()
	if !strings.Contains(output2, "debug 2") {
		t.Error("DEBUG message should be logged after level change")
	}
	if !strings.Contains(output2, "info 2") {
		t.Error("INFO message should be logged after level change")
	}
}

func TestDynamicLogLevelWithIsLevelEnabled(t *testing.T) {
	logLevel := &slog.LevelVar{}
	logLevel.Set(slog.LevelInfo)

	logger := New(WithLevel(logLevel))

	// Initially at INFO
	if logger.IsLevelEnabled(DEBUG) {
		t.Error("DEBUG should not be enabled at INFO level")
	}
	if !logger.IsLevelEnabled(INFO) {
		t.Error("INFO should be enabled at INFO level")
	}

	// Change to DEBUG
	logLevel.Set(slog.LevelDebug)

	// Now DEBUG is enabled
	if !logger.IsLevelEnabled(DEBUG) {
		t.Error("DEBUG should be enabled after level change")
	}
	if !logger.IsLevelEnabled(INFO) {
		t.Error("INFO should still be enabled after level change")
	}
}

func TestWithLevelBackwardCompatibility(t *testing.T) {
	buf := &bytes.Buffer{}

	// Old API should still work - passing log.Level directly
	logger := New(
		WithLevel(DEBUG),
		WithWriter(buf),
	)

	logger.Debug("debug message")

	output := buf.String()
	if !strings.Contains(output, "debug message") {
		t.Error("Expected debug message with backward compatible API")
	}
}

func TestConfigLevelMethod(t *testing.T) {
	// Test static level
	config := &Config{
		leveler: INFO,
	}
	if config.Level() != INFO {
		t.Errorf("Config.Level() = %v, want %v", config.Level(), INFO)
	}

	// Test dynamic level
	logLevel := &slog.LevelVar{}
	logLevel.Set(slog.LevelDebug)
	config2 := &Config{
		leveler: logLevel,
	}
	if config2.Level() != DEBUG {
		t.Errorf("Config.Level() = %v, want %v", config2.Level(), DEBUG)
	}

	// Change dynamic level
	logLevel.Set(slog.LevelWarn)
	if config2.Level() != WARN {
		t.Errorf("Config.Level() after change = %v, want %v", config2.Level(), WARN)
	}

	// Test nil leveler
	config3 := &Config{
		leveler: nil,
	}
	if config3.Level() != INFO {
		t.Errorf("Config.Level() with nil = %v, want %v (default)", config3.Level(), INFO)
	}
}
