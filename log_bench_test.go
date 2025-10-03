package log

import (
	"bytes"
	"log/slog"
	"testing"
)

func BenchmarkLoggerInfo(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := New().WithWriter(buf).Build()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "key", "value", "count", i)
	}
}

func BenchmarkSlogInfo(b *testing.B) {
	buf := &bytes.Buffer{}
	slogger := slog.New(slog.NewJSONHandler(buf, nil))
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slogger.Info("benchmark message", "key", "value", "count", i)
	}
}

func BenchmarkLoggerWithFields(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := New().WithWriter(buf).WithDefaultFields("app", "test", "version", "1.0").Build()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "key", "value")
	}
}

func BenchmarkLoggerConsole(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := New().WithWriter(buf).WithFormat(FormatConsole).Build()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "key", "value", "count", i)
	}
}

func BenchmarkLoggerJSON(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := New().WithWriter(buf).WithFormat(FormatJSON).Build()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "key", "value", "count", i)
	}
}

func BenchmarkLoggerText(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := New().WithWriter(buf).WithFormat(FormatText).Build()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "key", "value", "count", i)
	}
}

func BenchmarkLoggerDisabledLevel(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := New().WithWriter(buf).WithLevel(ERROR).Build()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Debug("benchmark message", "key", "value", "count", i)
	}
}

func BenchmarkPackageLevelInfo(b *testing.B) {
	buf := &bytes.Buffer{}
	SetDefault(New().WithWriter(buf).Build())
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark message", "key", "value", "count", i)
	}
}

func BenchmarkLoggerWithFieldsChain(b *testing.B) {
	buf := &bytes.Buffer{}
	baseLogger := New().WithWriter(buf).Build()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger := baseLogger.WithFields("request_id", i)
		logger.Info("benchmark message")
	}
}

func BenchmarkHook(b *testing.B) {
	buf := &bytes.Buffer{}
	hook := func(entry *Entry) error {
		return nil
	}
	logger := New().WithWriter(buf).WithHook(hook).Build()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "key", "value")
	}
}

func BenchmarkMultipleWriters(b *testing.B) {
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}
	logger := New().WithWriters(buf1, buf2).Build()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "key", "value")
	}
}
