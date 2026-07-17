package logger

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/v3rsionx/tg_bot/internal/constants"
)

// RotateOptions configures file rotation.
type RotateOptions struct {
	Filename   string
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
	Daily      bool
	Clock      func() time.Time
}

// RotatingFile is a thread-safe writer with size and/or daily rotation.
type RotatingFile struct {
	mu         sync.Mutex
	opts       RotateOptions
	file       *os.File
	size       int64
	day        string
	clock      func() time.Time
}

// NewRotatingFile opens (or creates) a rotating log file.
func NewRotatingFile(opts RotateOptions) (*RotatingFile, error) {
	if opts.Filename == "" {
		opts.Filename = constants.DefaultLogFileName
	}
	if opts.MaxSizeMB <= 0 {
		opts.MaxSizeMB = constants.DefaultLogMaxSizeMB
	}
	if opts.MaxBackups <= 0 {
		opts.MaxBackups = constants.DefaultLogMaxBackups
	}
	if opts.MaxAgeDays <= 0 {
		opts.MaxAgeDays = constants.DefaultLogMaxAgeDays
	}
	if opts.Clock == nil {
		opts.Clock = func() time.Time { return time.Now().UTC() }
	}
	r := &RotatingFile{opts: opts, clock: opts.Clock}
	if err := r.open(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *RotatingFile) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.rotateIfNeeded(len(p)); err != nil {
		return 0, err
	}
	n, err := r.file.Write(p)
	r.size += int64(n)
	return n, err
}

func (r *RotatingFile) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.file == nil {
		return nil
	}
	err := r.file.Close()
	r.file = nil
	return err
}

func (r *RotatingFile) open() error {
	if err := os.MkdirAll(filepath.Dir(r.opts.Filename), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(r.opts.Filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}
	r.file = f
	r.size = info.Size()
	r.day = r.clock().Format("2006-01-02")
	return nil
}

func (r *RotatingFile) rotateIfNeeded(incoming int) error {
	now := r.clock()
	day := now.Format("2006-01-02")
	maxBytes := int64(r.opts.MaxSizeMB) * 1024 * 1024
	needDaily := r.opts.Daily && day != r.day
	needSize := r.size+int64(incoming) > maxBytes && r.size > 0
	if !needDaily && !needSize {
		return nil
	}
	return r.rotate(now)
}

func (r *RotatingFile) rotate(now time.Time) error {
	if r.file != nil {
		_ = r.file.Close()
		r.file = nil
	}
	ts := now.Format("20060102-150405")
	backup := fmt.Sprintf("%s.%s", r.opts.Filename, ts)
	if err := os.Rename(r.opts.Filename, backup); err != nil && !os.IsNotExist(err) {
		return err
	}
	if r.opts.Compress {
		_ = compressFile(backup)
	}
	_ = r.cleanup()
	return r.open()
}

func compressFile(path string) error {
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()
	outPath := path + ".gz"
	out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	zw := gzip.NewWriter(out)
	_, copyErr := io.Copy(zw, in)
	closeErr := zw.Close()
	outErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if outErr != nil {
		return outErr
	}
	return os.Remove(path)
}

func (r *RotatingFile) cleanup() error {
	dir := filepath.Dir(r.opts.Filename)
	base := filepath.Base(r.opts.Filename)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	type item struct {
		path string
		mod  time.Time
	}
	var backups []item
	prefix := base + "."
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		backups = append(backups, item{path: filepath.Join(dir, name), mod: info.ModTime()})
	}
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].mod.After(backups[j].mod)
	})
	cutoff := r.clock().AddDate(0, 0, -r.opts.MaxAgeDays)
	for i, b := range backups {
		if i >= r.opts.MaxBackups || b.mod.Before(cutoff) {
			_ = os.Remove(b.path)
		}
	}
	return nil
}
