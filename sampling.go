package log

import (
	"io"
	"sync"
	"sync/atomic"
)

// samplingWriter implements log sampling.
type samplingWriter struct {
	writer  io.Writer
	config  *SamplingConfig
	counter atomic.Uint64
	mu      sync.Mutex
}

// newSamplingWriter creates a new sampling writer.
func newSamplingWriter(w io.Writer, config *SamplingConfig) io.Writer {
	return &samplingWriter{
		writer: w,
		config: config,
	}
}

// Write writes data with sampling applied.
func (sw *samplingWriter) Write(p []byte) (n int, err error) {
	count := sw.counter.Add(1)

	// Log first N messages
	if count <= uint64(sw.config.First) {
		return sw.writer.Write(p)
	}

	// Then log every Nth message
	if sw.config.Thereafter > 0 {
		if (count-uint64(sw.config.First))%uint64(sw.config.Thereafter) == 0 {
			return sw.writer.Write(p)
		}
	}

	// Pretend we wrote it
	return len(p), nil
}
