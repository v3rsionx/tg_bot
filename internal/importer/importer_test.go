package importer_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/v3rsionx/tg_bot/internal/importer"
)

// TestImporterReadsConverterStandardCSV maps id,name,phone,username,extras.
func TestImporterReadsConverterStandardCSV(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "standard.csv")
	content := "id,name,phone,username,extras\n" +
		`6473397867,"Fabiana Umbelino",+15551110001,fabiana,"{""access_hash"":""81293"",""country"":""BR""}"` + "\n" +
		"1002,Ana Silva,+15551110002,ana,{}\n"
	if err := os.WriteFile(source, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	idStore := newMemoryEngine()
	phoneStore := newMemoryEngine()
	usernameStore := newMemoryEngine()

	im, err := importer.New(importer.Config{
		Sources:   []string{source},
		Delimiter: ',',
		Workers:   1,
		BatchSize: 10,
	}, importer.Stores{
		ID:       idStore,
		Phone:    phoneStore,
		Username: usernameStore,
	}, importer.NopLogger{}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	stats, err := im.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if stats.Inserts != 2 {
		t.Fatalf("Inserts = %d, want 2", stats.Inserts)
	}
	if stats.ExtrasRetained != 1 {
		t.Fatalf("ExtrasRetained = %d, want 1 (non-empty extras row)", stats.ExtrasRetained)
	}
	gotPhone, err := phoneStore.Get(context.Background(), []byte("+15551110001"))
	if err != nil || string(gotPhone) != "6473397867" {
		t.Fatalf("phone lookup = %q err=%v", gotPhone, err)
	}
	gotUser, err := usernameStore.Get(context.Background(), []byte("fabiana"))
	if err != nil || string(gotUser) != "6473397867" {
		t.Fatalf("username lookup = %q err=%v", gotUser, err)
	}
	payload, err := idStore.Get(context.Background(), []byte("6473397867"))
	if err != nil {
		t.Fatalf("id get: %v", err)
	}
	wantPayload := "+15551110001\x00fabiana\x00Fabiana Umbelino\x00" +
		`{"access_hash":"81293","country":"BR"}`
	if string(payload) != wantPayload {
		t.Fatalf("payload = %q, want %q", payload, wantPayload)
	}
}

// TestImporterAcceptsOptionalPhoneUsernameCombinations covers ID-centric rows.
func TestImporterAcceptsOptionalPhoneUsernameCombinations(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "optional.csv")
	content := "id,name,phone,username,extras\n" +
		"1001,,,,\n" + // id only
		"1002,Only Name,,,\n" + // id + name
		`1003,,,,"{""access_hash"":""1"}"` + "\n" + // id + extras
		"1004,,+15551110004,,\n" + // id + phone
		"1005,,,alice_only,\n" + // id + username
		"1006,,+15551110006,both_user,\n" // id + phone + username
	if err := os.WriteFile(source, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	idStore := newMemoryEngine()
	phoneStore := newMemoryEngine()
	usernameStore := newMemoryEngine()

	im, err := importer.New(importer.Config{
		Sources:   []string{source},
		Delimiter: ',',
		Workers:   1,
		BatchSize: 10,
	}, importer.Stores{
		ID:       idStore,
		Phone:    phoneStore,
		Username: usernameStore,
	}, importer.NopLogger{}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	stats, err := im.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if stats.Inserts != 6 {
		t.Fatalf("Inserts = %d, want 6; invalid=%d", stats.Inserts, stats.RecordsInvalid)
	}
	if stats.RecordsInvalid != 0 {
		t.Fatalf("RecordsInvalid = %d, want 0", stats.RecordsInvalid)
	}
	if stats.ExtrasRetained != 1 {
		t.Fatalf("ExtrasRetained = %d, want 1", stats.ExtrasRetained)
	}

	for _, id := range []string{"1001", "1002", "1003", "1004", "1005", "1006"} {
		if _, err := idStore.Get(context.Background(), []byte(id)); err != nil {
			t.Fatalf("missing id %s: %v", id, err)
		}
	}
	// Empty phone/username must not create reverse indexes.
	if phoneStore.Len() != 2 || usernameStore.Len() != 2 {
		t.Fatalf("phone/username sizes = %d/%d, want 2/2", phoneStore.Len(), usernameStore.Len())
	}
	gotPhone, err := phoneStore.Get(context.Background(), []byte("+15551110004"))
	if err != nil || string(gotPhone) != "1004" {
		t.Fatalf("phone 1004 = %q err=%v", gotPhone, err)
	}
	gotUser, err := usernameStore.Get(context.Background(), []byte("alice_only"))
	if err != nil || string(gotUser) != "1005" {
		t.Fatalf("username 1005 = %q err=%v", gotUser, err)
	}
	payload, err := idStore.Get(context.Background(), []byte("1001"))
	if err != nil {
		t.Fatalf("id 1001 get: %v", err)
	}
	if string(payload) != "\x00\x00\x00{}" {
		t.Fatalf("id-only payload = %q, want phone\\0username\\0name\\0{}", payload)
	}
	namePayload, err := idStore.Get(context.Background(), []byte("1002"))
	if err != nil {
		t.Fatalf("id 1002 get: %v", err)
	}
	if string(namePayload) != "\x00\x00Only Name\x00{}" {
		t.Fatalf("name payload = %q", namePayload)
	}
	extrasPayload, err := idStore.Get(context.Background(), []byte("1003"))
	if err != nil {
		t.Fatalf("id 1003 get: %v", err)
	}
	if string(extrasPayload) != "\x00\x00\x00"+`{"access_hash":"1"}` {
		t.Fatalf("extras payload = %q", extrasPayload)
	}
}

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
		BatchSize:      1, // flush each row so Exists sees prior writes
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
