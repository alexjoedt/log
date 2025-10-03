package log

import (
	"bufio"
	"io"
	"sync"
	"time"
)

// bufferedWriter implements buffered writing with auto-flush.
type bufferedWriter struct {
	writer       io.Writer
	config       *BufferConfig
	buffer       *bufio.Writer
	ticker       *time.Ticker
	mu           sync.Mutex
	stopChan     chan struct{}
	flushOnError bool
}

// newBufferedWriter creates a new buffered writer.
func newBufferedWriter(w io.Writer, config *BufferConfig) io.Writer {
	bw := &bufferedWriter{
		writer:       w,
		config:       config,
		buffer:       bufio.NewWriterSize(w, config.Size),
		stopChan:     make(chan struct{}),
		flushOnError: true,
	}

	// Start auto-flush ticker if interval is set
	if config.FlushInterval > 0 {
		bw.ticker = time.NewTicker(config.FlushInterval)
		go bw.autoFlush()
	}

	return bw
}

// Write writes data to the buffer.
func (bw *bufferedWriter) Write(p []byte) (n int, err error) {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	n, err = bw.buffer.Write(p)

	// Auto-flush on error or fatal logs
	// This is a simple heuristic - in production you'd want better detection
	if bw.flushOnError && (err != nil || containsErrorKeyword(p)) {
		_ = bw.buffer.Flush()
	}

	return n, err
}

// autoFlush periodically flushes the buffer.
func (bw *bufferedWriter) autoFlush() {
	for {
		select {
		case <-bw.ticker.C:
			bw.mu.Lock()
			_ = bw.buffer.Flush()
			bw.mu.Unlock()
		case <-bw.stopChan:
			return
		}
	}
}

// Flush flushes the buffer.
func (bw *bufferedWriter) Flush() error {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	return bw.buffer.Flush()
}

// Close closes the buffered writer.
func (bw *bufferedWriter) Close() error {
	if bw.ticker != nil {
		bw.ticker.Stop()
	}
	close(bw.stopChan)

	bw.mu.Lock()
	defer bw.mu.Unlock()
	return bw.buffer.Flush()
}

// containsErrorKeyword checks if the log message contains error keywords.
func containsErrorKeyword(p []byte) bool {
	// Simple check for ERROR or FATAL in the message
	keywords := [][]byte{
		[]byte("ERROR"),
		[]byte("FATAL"),
		[]byte("[ERROR]"),
		[]byte("[FATAL]"),
	}

	for _, keyword := range keywords {
		if contains(p, keyword) {
			return true
		}
	}
	return false
}

// contains checks if a byte slice contains a substring.
func contains(haystack, needle []byte) bool {
	if len(needle) == 0 {
		return true
	}
	if len(haystack) < len(needle) {
		return false
	}

	for i := 0; i <= len(haystack)-len(needle); i++ {
		found := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				found = false
				break
			}
		}
		if found {
			return true
		}
	}
	return false
}
