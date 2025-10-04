package log

import (
	"bytes"
	"sync"
)

// TestLogger is a logger designed for testing that captures log entries.
type TestLogger struct {
	*Logger
	hook *TestHook
}

// TestHook captures log entries for testing.
type TestHook struct {
	entries []*Entry
	mu      sync.RWMutex
}

// NewTestLogger creates a logger and hook for testing.
func NewTestLogger() (*TestLogger, *TestHook) {
	hook := &TestHook{
		entries: make([]*Entry, 0),
	}

	buf := &bytes.Buffer{}
	logger := New(
		WithWriter(buf),
		WithLevel(TRACE),
		WithHook(hook.capture),
	)

	return &TestLogger{
		Logger: logger,
		hook:   hook,
	}, hook
}

// capture is the hook function that captures entries.
func (h *TestHook) capture(entry *Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Deep copy the entry
	entryCopy := &Entry{
		Level:   entry.Level,
		Message: entry.Message,
		Time:    entry.Time,
		Fields:  make([]any, len(entry.Fields)),
		Error:   entry.Error,
	}
	copy(entryCopy.Fields, entry.Fields)

	h.entries = append(h.entries, entryCopy)
	return nil
}

// Entries returns all captured entries.
func (h *TestHook) Entries() []*Entry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	entries := make([]*Entry, len(h.entries))
	copy(entries, h.entries)
	return entries
}

// LastEntry returns the last captured entry, or nil if none.
func (h *TestHook) LastEntry() *Entry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.entries) == 0 {
		return nil
	}
	return h.entries[len(h.entries)-1]
}

// Reset clears all captured entries.
func (h *TestHook) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = make([]*Entry, 0)
}

// Count returns the number of captured entries.
func (h *TestHook) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.entries)
}

// CountLevel returns the number of entries at the given level.
func (h *TestHook) CountLevel(level Level) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for _, entry := range h.entries {
		if entry.Level == level {
			count++
		}
	}
	return count
}

// HasMessage checks if any entry contains the given message.
func (h *TestHook) HasMessage(msg string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, entry := range h.entries {
		if entry.Message == msg {
			return true
		}
	}
	return false
}

// HasLevel checks if any entry has the given level.
func (h *TestHook) HasLevel(level Level) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, entry := range h.entries {
		if entry.Level == level {
			return true
		}
	}
	return false
}
