package log

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// rotatingWriter implements log rotation based on size, age, and backup count.
type rotatingWriter struct {
	writer      io.Writer
	config      *RotationConfig
	currentFile *os.File
	currentSize int64
	mu          sync.Mutex
	filename    string
}

// newRotatingWriter creates a new rotating writer.
func newRotatingWriter(w io.Writer, config *RotationConfig) io.Writer {
	// If the writer is not a file, we can't rotate
	file, ok := w.(*os.File)
	if !ok {
		return w
	}

	return &rotatingWriter{
		writer:      w,
		config:      config,
		currentFile: file,
		filename:    file.Name(),
	}
}

// Write writes data and rotates if necessary.
func (rw *rotatingWriter) Write(p []byte) (n int, err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	// Check if rotation is needed
	if rw.shouldRotate(len(p)) {
		if err := rw.rotate(); err != nil {
			return 0, err
		}
	}

	n, err = rw.currentFile.Write(p)
	if err == nil {
		rw.currentSize += int64(n)
	}
	return n, err
}

// shouldRotate checks if rotation is needed.
func (rw *rotatingWriter) shouldRotate(writeSize int) bool {
	if rw.config.MaxSize <= 0 {
		return false
	}

	maxBytes := int64(rw.config.MaxSize) * 1024 * 1024
	return rw.currentSize+int64(writeSize) > maxBytes
}

// rotate performs the log rotation.
func (rw *rotatingWriter) rotate() error {
	// Close current file
	if rw.currentFile != nil {
		rw.currentFile.Close()
	}

	// Rename current file with timestamp
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	rotatedName := fmt.Sprintf("%s.%s", rw.filename, timestamp)

	if err := os.Rename(rw.filename, rotatedName); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Compress if configured
	if rw.config.Compress {
		if err := rw.compressFile(rotatedName); err != nil {
			// Log error but don't fail
			_ = err
		}
	}

	// Clean up old files
	rw.cleanupOldFiles()

	// Open new file
	f, err := os.OpenFile(rw.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	rw.currentFile = f
	rw.currentSize = 0
	return nil
}

// compressFile compresses a log file.
func (rw *rotatingWriter) compressFile(filename string) error {
	src, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(filename + ".gz")
	if err != nil {
		return err
	}
	defer dst.Close()

	gzw := gzip.NewWriter(dst)
	defer gzw.Close()

	if _, err := io.Copy(gzw, src); err != nil {
		return err
	}

	// Remove original file
	return os.Remove(filename)
}

// cleanupOldFiles removes old log files based on MaxBackups and MaxAge.
func (rw *rotatingWriter) cleanupOldFiles() {
	dir := filepath.Dir(rw.filename)
	base := filepath.Base(rw.filename)

	files, err := filepath.Glob(filepath.Join(dir, base+".*"))
	if err != nil {
		return
	}

	// Sort files by modification time
	sort.Slice(files, func(i, j int) bool {
		fi, _ := os.Stat(files[i])
		fj, _ := os.Stat(files[j])
		return fi.ModTime().After(fj.ModTime())
	})

	// Remove files exceeding MaxBackups
	if rw.config.MaxBackups > 0 && len(files) > rw.config.MaxBackups {
		for _, f := range files[rw.config.MaxBackups:] {
			os.Remove(f)
		}
		files = files[:rw.config.MaxBackups]
	}

	// Remove files exceeding MaxAge
	if rw.config.MaxAge > 0 {
		cutoff := time.Now().AddDate(0, 0, -rw.config.MaxAge)
		for _, f := range files {
			info, err := os.Stat(f)
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				os.Remove(f)
			}
		}
	}
}
