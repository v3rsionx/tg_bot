package lmdb_test

import (
	"context"
	"errors"
	"testing"

	"github.com/v3rsi/tgbot-versionx/internal/database/lmdb"
)

// TestDBCRUDAndBatch covers core engine operations.
func TestDBCRUDAndBatch(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	if err := db.Put(ctx, []byte("alpha"), []byte("1")); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	got, err := db.Get(ctx, []byte("alpha"))
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if string(got) != "1" {
		t.Fatalf("Get() = %q, want %q", got, "1")
	}

	exists, err := db.Exists(ctx, []byte("alpha"))
	if err != nil || !exists {
		t.Fatalf("Exists() = (%v, %v), want (true, nil)", exists, err)
	}

	if err := db.BatchPut(ctx, []lmdb.KeyValue{
		{Key: []byte("beta"), Value: []byte("2")},
		{Key: []byte("gamma"), Value: []byte("3")},
	}); err != nil {
		t.Fatalf("BatchPut() error = %v", err)
	}

	if err := db.BatchDelete(ctx, [][]byte{[]byte("beta")}); err != nil {
		t.Fatalf("BatchDelete() error = %v", err)
	}
	exists, err = db.Exists(ctx, []byte("beta"))
	if err != nil || exists {
		t.Fatalf("Exists(beta) = (%v, %v), want (false, nil)", exists, err)
	}

	if err := db.Delete(ctx, []byte("alpha")); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := db.Get(ctx, []byte("alpha")); !errors.Is(err, lmdb.ErrNotFound) {
		t.Fatalf("Get(missing) error = %v, want ErrNotFound", err)
	}

	if err := db.Sync(ctx, true); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	stats, err := db.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.Entries == 0 {
		t.Fatal("Stats().Entries = 0, want remaining records")
	}
	if stats.MapSize == 0 {
		t.Fatal("Stats().MapSize = 0, want configured map size")
	}
}

// TestDBReaderWriterAndCursor covers transactions and iteration.
func TestDBReaderWriterAndCursor(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	writer, err := db.Writer(ctx)
	if err != nil {
		t.Fatalf("Writer() error = %v", err)
	}
	if err := writer.Put(ctx, []byte("a"), []byte("1")); err != nil {
		t.Fatalf("writer.Put(a) error = %v", err)
	}
	if err := writer.Put(ctx, []byte("b"), []byte("2")); err != nil {
		t.Fatalf("writer.Put(b) error = %v", err)
	}
	if err := writer.Commit(ctx); err != nil {
		t.Fatalf("writer.Commit() error = %v", err)
	}

	reader, err := db.Reader(ctx)
	if err != nil {
		t.Fatalf("Reader() error = %v", err)
	}
	defer func() { _ = reader.Abort() }()

	if !reader.Readonly() {
		t.Fatal("Reader().Readonly() = false, want true")
	}
	if err := reader.Put(ctx, []byte("c"), []byte("3")); !errors.Is(err, lmdb.ErrReadOnly) {
		t.Fatalf("reader.Put() error = %v, want ErrReadOnly", err)
	}

	cursor, err := reader.Cursor(ctx)
	if err != nil {
		t.Fatalf("Cursor() error = %v", err)
	}
	defer func() { _ = cursor.Close() }()

	key, value, err := cursor.First(ctx)
	if err != nil {
		t.Fatalf("First() error = %v", err)
	}
	if string(key) != "a" || string(value) != "1" {
		t.Fatalf("First() = (%q, %q), want (a, 1)", key, value)
	}

	key, value, err = cursor.Next(ctx)
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if string(key) != "b" || string(value) != "2" {
		t.Fatalf("Next() = (%q, %q), want (b, 2)", key, value)
	}

	key, value, err = cursor.Seek(ctx, []byte("b"))
	if err != nil {
		t.Fatalf("Seek() error = %v", err)
	}
	if string(key) != "b" || string(value) != "2" {
		t.Fatalf("Seek() = (%q, %q), want (b, 2)", key, value)
	}
}

// TestDBAutoMapResizeRetriesAfterMapFull verifies automatic growth.
func TestDBAutoMapResizeRetriesAfterMapFull(t *testing.T) {
	ctx := context.Background()
	db, err := lmdb.OpenDB(ctx, lmdb.Config{
		Path:           t.TempDir(),
		InitialMapSize: 8 << 10,
		MaxMapSize:     64 << 20,
		MapGrowth:      1 << 20,
		MaxReaders:     64,
	})
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	value := make([]byte, 4<<10)
	for i := 0; i < 64; i++ {
		key := []byte{byte(i >> 8), byte(i)}
		if err := db.Put(ctx, key, value); err != nil {
			t.Fatalf("Put(%d) error = %v", i, err)
		}
	}

	stats, err := db.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.MapSize <= 8<<10 {
		t.Fatalf("MapSize = %d, want growth above initial size", stats.MapSize)
	}
}

// openTestDB constructs an isolated LMDB engine for tests.
func openTestDB(t *testing.T) *lmdb.DB {
	t.Helper()

	db, err := lmdb.OpenDB(context.Background(), lmdb.Config{
		Path:           t.TempDir(),
		InitialMapSize: 32 << 20,
		MaxMapSize:     256 << 20,
		MapGrowth:      16 << 20,
		MaxReaders:     128,
	})
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})
	return db
}
