package importer_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/v3rsionx/tg_bot/internal/importer"
)

// TestImporterStreamsCSVAndWritesExactIndexes covers the happy path.
func TestImporterStreamsCSVAndWritesExactIndexes(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "sample.csv")
	content := "id,phone,username\n" +
		"1001,+15551110001,alice_one\n" +
		"1002,+15551110002,bob_two\n" +
		"bad-id,+15551110003,charlie\n" +
		"1001,+15551110099,alice_dup\n" +
		"1003,+15551110003,carol_three\n"
	if err := os.WriteFile(source, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	idStore := newMemoryEngine()
	phoneStore := newMemoryEngine()
	usernameStore := newMemoryEngine()

	var last importer.Progress
	im, err := importer.New(importer.Config{
		Sources:          []string{source},
		Delimiter:        ',',
		HasHeader:        true,
		Workers:          2,
		BatchSize:        2,
		QueueSize:        8,
		SkipDuplicateIDs: true,
		ProgressInterval: 50 * time.Millisecond,
	}, importer.Stores{
		ID:       idStore,
		Phone:    phoneStore,
		Username: usernameStore,
	}, importer.NopLogger{}, func(p importer.Progress) {
		last = p
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	stats, err := im.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if stats.Inserts != 3 {
		t.Fatalf("Inserts = %d, want 3", stats.Inserts)
	}
	if stats.Duplicates != 1 {
		t.Fatalf("Duplicates = %d, want 1", stats.Duplicates)
	}
	if stats.RecordsInvalid < 1 {
		t.Fatalf("RecordsInvalid = %d, want at least 1", stats.RecordsInvalid)
	}
	if idStore.Len() != 3 || phoneStore.Len() != 3 || usernameStore.Len() != 3 {
		t.Fatalf("store sizes = id:%d phone:%d username:%d", idStore.Len(), phoneStore.Len(), usernameStore.Len())
	}
	if last.Statistics.Inserts != stats.Inserts {
		t.Fatalf("progress inserts = %d, want %d", last.Statistics.Inserts, stats.Inserts)
	}
}

// TestImporterResumeSkipsProcessedBytes verifies checkpoint resume behavior.
func TestImporterResumeSkipsProcessedBytes(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "resume.txt")
	content := "2001|+15552220001|user_one\n2002|+15552220002|user_two\n"
	if err := os.WriteFile(source, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	checkpoint := filepath.Join(dir, "checkpoint.json")
	stores := importer.Stores{
		ID:       newMemoryEngine(),
		Phone:    newMemoryEngine(),
		Username: newMemoryEngine(),
	}

	first, err := importer.New(importer.Config{
		Sources:          []string{source},
		Delimiter:        '|',
		Workers:          1,
		BatchSize:        1,
		Resume:           true,
		CheckpointPath:   checkpoint,
		SkipDuplicateIDs: true,
	}, stores, importer.NopLogger{}, nil)
	if err != nil {
		t.Fatalf("New(first) error = %v", err)
	}
	if _, err := first.Run(context.Background()); err != nil {
		t.Fatalf("Run(first) error = %v", err)
	}

	// Append a new row and resume; existing IDs should be treated as duplicates.
	file, err := os.OpenFile(source, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	if _, err := file.WriteString("2003|+15552220003|user_three\n"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	_ = file.Close()

	second, err := importer.New(importer.Config{
		Sources:          []string{source},
		Delimiter:        '|',
		Workers:          1,
		BatchSize:        1,
		Resume:           true,
		CheckpointPath:   checkpoint,
		SkipDuplicateIDs: true,
	}, stores, importer.NopLogger{}, nil)
	if err != nil {
		t.Fatalf("New(second) error = %v", err)
	}
	stats, err := second.Run(context.Background())
	if err != nil {
		t.Fatalf("Run(second) error = %v", err)
	}
	if stats.Inserts != 1 {
		t.Fatalf("resume Inserts = %d, want 1", stats.Inserts)
	}
	if stores.ID.(*memoryEngine).Len() != 3 {
		t.Fatalf("id store len = %d, want 3", stores.ID.(*memoryEngine).Len())
	}
}

// TestImporterRespectsContextCancellation verifies graceful shutdown.
func TestImporterRespectsContextCancellation(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "large.csv")
	f, err := os.Create(source)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	for i := 0; i < 1000; i++ {
		if _, err := f.WriteString("3001,+15553330001,user_cancel\n"); err != nil {
			t.Fatalf("WriteString() error = %v", err)
		}
	}
	_ = f.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	im, err := importer.New(importer.Config{
		Sources:   []string{source},
		Delimiter: ',',
		Workers:   2,
		BatchSize: 10,
	}, importer.Stores{
		ID:       newMemoryEngine(),
		Phone:    newMemoryEngine(),
		Username: newMemoryEngine(),
	}, importer.NopLogger{}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if _, err := im.Run(ctx); err == nil {
		t.Fatal("Run() error = nil, want cancellation error")
	}
}

// TestImporterUpdateExistingRewritesIndexes covers update statistics.
func TestImporterUpdateExistingRewritesIndexes(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "update.csv")
	content := "4001,+15554440001,old_user\n4001,+15554440099,new_user\n"
	if err := os.WriteFile(source, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	im, err := importer.New(importer.Config{
		Sources:        []string{source},
		Delimiter:      ',',
		Workers:        1,
		BatchSize:      10,
		UpdateExisting: true,
	}, importer.Stores{
		ID:       newMemoryEngine(),
		Phone:    newMemoryEngine(),
		Username: newMemoryEngine(),
	}, importer.NopLogger{}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	stats, err := im.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if stats.Inserts != 1 || stats.Updates != 1 {
		t.Fatalf("Inserts=%d Updates=%d, want 1 and 1", stats.Inserts, stats.Updates)
	}
}
