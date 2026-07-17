package converter

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger is the injectable logging contract.
type Logger interface {
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
}

// NopLogger discards logs.
type NopLogger struct{}

func (NopLogger) Infof(format string, args ...any)  {}
func (NopLogger) Warnf(format string, args ...any)  {}
func (NopLogger) Errorf(format string, args ...any) {}

// SkipLogger appends skipped-row details to a log file.
type SkipLogger struct {
	mu   sync.Mutex
	path string
	w    io.WriteCloser
}

// NewSkipLogger opens (or creates) the skip log file.
func NewSkipLogger(path string) (*SkipLogger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, wrap("create log dir", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, wrap("open skip log", err)
	}
	return &SkipLogger{path: path, w: f}, nil
}

// LogSkip records a skipped row.
func (l *SkipLogger) LogSkip(file string, line uint64, reason string) error {
	if l == nil || l.w == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_, err := fmt.Fprintf(l.w, "%s file=%s line=%d reason=%s\n",
		time.Now().UTC().Format(time.RFC3339), file, line, reason)
	return err
}

// Close closes the underlying file.
func (l *SkipLogger) Close() error {
	if l == nil || l.w == nil {
		return nil
	}
	return l.w.Close()
}
