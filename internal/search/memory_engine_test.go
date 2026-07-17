package search_test

import (
	"context"
	"sync"

	"github.com/v3rsionx/tg_bot/internal/database/lmdb"
)

// memoryEngine is an in-memory lmdb.Engine for search unit tests.
type memoryEngine struct {
	mu   sync.RWMutex
	data map[string][]byte
}

// newMemoryEngine constructs an empty memory engine.
func newMemoryEngine() *memoryEngine {
	return &memoryEngine{data: make(map[string][]byte)}
}

// Open is a no-op for the in-memory engine.
func (m *memoryEngine) Open(ctx context.Context) error { return ctx.Err() }

// Close is a no-op for the in-memory engine.
func (m *memoryEngine) Close() error { return nil }

// Put stores one key/value pair.
func (m *memoryEngine) Put(ctx context.Context, key, value []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[string(key)] = append([]byte(nil), value...)
	return nil
}

// Get returns a copy of the value for key.
func (m *memoryEngine) Get(ctx context.Context, key []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.data[string(key)]
	if !ok {
		return nil, lmdb.ErrNotFound
	}
	return append([]byte(nil), value...), nil
}

// Delete removes key when present.
func (m *memoryEngine) Delete(ctx context.Context, key []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[string(key)]; !ok {
		return lmdb.ErrNotFound
	}
	delete(m.data, string(key))
	return nil
}

// Exists reports whether key is present.
func (m *memoryEngine) Exists(ctx context.Context, key []byte) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.data[string(key)]
	return ok, nil
}

// BatchPut stores many key/value pairs.
func (m *memoryEngine) BatchPut(ctx context.Context, items []lmdb.KeyValue) error {
	for _, item := range items {
		if err := m.Put(ctx, item.Key, item.Value); err != nil {
			return err
		}
	}
	return nil
}

// BatchDelete removes many keys.
func (m *memoryEngine) BatchDelete(ctx context.Context, keys [][]byte) error {
	for _, key := range keys {
		_ = m.Delete(ctx, key)
	}
	return nil
}

// Stats returns a minimal stats snapshot.
func (m *memoryEngine) Stats(ctx context.Context) (lmdb.Stats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return lmdb.Stats{Entries: uint64(len(m.data))}, ctx.Err()
}

// Sync is a no-op for the in-memory engine.
func (m *memoryEngine) Sync(ctx context.Context, force bool) error { return ctx.Err() }

// Reader is unused by search tests.
func (m *memoryEngine) Reader(ctx context.Context) (lmdb.Txn, error) {
	return nil, lmdb.ErrClosed
}

// Writer is unused by search tests.
func (m *memoryEngine) Writer(ctx context.Context) (lmdb.Txn, error) {
	return nil, lmdb.ErrClosed
}

var _ lmdb.Engine = (*memoryEngine)(nil)
