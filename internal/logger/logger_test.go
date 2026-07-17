package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestConsoleAndJSONLogger(t *testing.T) {
	var buf bytes.Buffer
	log := New(Options{Level: LevelDebug, CorrelationID: "cid-1", FatalFunc: func(int) {}}, NewConsoleSink(&buf))
	log.Info("hello", Fields{"k": "v"})
	if !strings.Contains(buf.String(), "hello") || !strings.Contains(buf.String(), "cid-1") {
		t.Fatalf("console output = %q", buf.String())
	}

	buf.Reset()
	jlog := NewJSONLogger(LevelInfo, &buf)
	jlog.Warn("json-msg", Fields{"n": 1})
	var entry Entry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if entry.Level != "warn" || entry.Message != "json-msg" {
		t.Fatalf("entry = %#v", entry)
	}
}

func TestContextCorrelationAndSpecialized(t *testing.T) {
	var buf bytes.Buffer
	base := New(Options{Level: LevelDebug}, NewConsoleSink(&buf))
	ctx := ContextWithCorrelationID(context.Background(), "corr-9")
	base.WithContext(ctx).Info("with-ctx")
	if !strings.Contains(buf.String(), "corr-9") {
		t.Fatalf("missing correlation id: %q", buf.String())
	}

	buf.Reset()
	NewSearchLogger(base).Query(7, "id", "123", true, 2*time.Millisecond)
	if !strings.Contains(buf.String(), "search.query") {
		t.Fatalf("search log missing: %q", buf.String())
	}
	buf.Reset()
	NewAdminLogger(base).Action(1, "ban", Fields{"target": 2})
	if !strings.Contains(buf.String(), "admin.action") {
		t.Fatalf("admin log missing: %q", buf.String())
	}
	buf.Reset()
	NewPerformanceLogger(base).Timing("sqlite.query", time.Millisecond, nil)
	if !strings.Contains(buf.String(), "performance.timing") {
		t.Fatalf("perf log missing: %q", buf.String())
	}
}

func TestFileRotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	rot, err := NewRotatingFile(RotateOptions{
		Filename:   path,
		MaxSizeMB:  1,
		MaxBackups: 3,
		MaxAgeDays: 7,
		Compress:   true,
		Daily:      true,
		Clock:      func() time.Time { return time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("NewRotatingFile: %v", err)
	}
	if _, err := rot.Write([]byte("line1\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	// Force rotate by switching day.
	rot.clock = func() time.Time { return time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC) }
	if _, err := rot.Write([]byte("line2\n")); err != nil {
		t.Fatalf("Write after day change: %v", err)
	}
	_ = rot.Close()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) < 2 {
		t.Fatalf("expected rotated files, got %d", len(entries))
	}
}

func TestParseLevel(t *testing.T) {
	if ParseLevel("DEBUG") != LevelDebug {
		t.Fatal("ParseLevel DEBUG")
	}
	if ParseLevel("nope") != LevelInfo {
		t.Fatal("ParseLevel default")
	}
}
