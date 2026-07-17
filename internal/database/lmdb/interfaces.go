package lmdb

import "context"

// Engine is the injectable LMDB storage contract.
type Engine interface {
	// Open initializes the LMDB environment and named database.
	Open(ctx context.Context) error
	// Close releases environment resources.
	Close() error
	// Put stores a single key/value pair.
	Put(ctx context.Context, key, value []byte) error
	// Get returns a copy of the value for key.
	Get(ctx context.Context, key []byte) ([]byte, error)
	// Delete removes key when it exists.
	Delete(ctx context.Context, key []byte) error
	// Exists reports whether key is present.
	Exists(ctx context.Context, key []byte) (bool, error)
	// BatchPut stores many key/value pairs in one write transaction.
	BatchPut(ctx context.Context, items []KeyValue) error
	// BatchDelete removes many keys in one write transaction.
	BatchDelete(ctx context.Context, keys [][]byte) error
	// Stats returns environment and database statistics.
	Stats(ctx context.Context) (Stats, error)
	// Sync flushes data to durable storage.
	Sync(ctx context.Context, force bool) error
	// Reader begins a read-only transaction.
	Reader(ctx context.Context) (Txn, error)
	// Writer begins a read-write transaction.
	Writer(ctx context.Context) (Txn, error)
}

// Txn is an LMDB transaction handle.
type Txn interface {
	// Get returns a copy of the value for key.
	Get(ctx context.Context, key []byte) ([]byte, error)
	// Put stores a key/value pair.
	Put(ctx context.Context, key, value []byte) error
	// Delete removes key when it exists.
	Delete(ctx context.Context, key []byte) error
	// Exists reports whether key is present.
	Exists(ctx context.Context, key []byte) (bool, error)
	// Cursor opens a cursor over the named database.
	Cursor(ctx context.Context) (Cursor, error)
	// Commit persists a read-write transaction.
	Commit(ctx context.Context) error
	// Abort discards the transaction.
	Abort() error
	// Readonly reports whether the transaction allows writes.
	Readonly() bool
}

// Cursor iterates keys inside an active transaction.
type Cursor interface {
	// First positions the cursor at the first key/value pair.
	First(ctx context.Context) (key, value []byte, err error)
	// Next advances the cursor to the next key/value pair.
	Next(ctx context.Context) (key, value []byte, err error)
	// Seek positions the cursor at the first key greater than or equal to key.
	Seek(ctx context.Context, key []byte) (foundKey, value []byte, err error)
	// Delete removes the current key/value pair.
	Delete(ctx context.Context) error
	// Close releases cursor resources.
	Close() error
}

// KeyValue is a single batch write item.
type KeyValue struct {
	Key   []byte
	Value []byte
}

// Stats captures LMDB environment and database metrics.
type Stats struct {
	MapSize       int64
	LastPageNo    int64
	LastTxnID     int64
	MaxReaders    uint
	NumReaders    uint
	Entries       uint64
	Depth         uint
	BranchPages   uint64
	LeafPages     uint64
	OverflowPages uint64
	PageSize      uint
}
