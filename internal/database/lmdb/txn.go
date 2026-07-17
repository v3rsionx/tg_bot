package lmdb

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	rawlmdb "github.com/bmatsuo/lmdb-go/lmdb"
)

// txn wraps an LMDB transaction and satisfies the Txn interface.
type txn struct {
	db         *DB
	raw        *rawlmdb.Txn
	readonly   bool
	holdsWrite bool
	mu         sync.Mutex
	closed     bool
}

// newTxn constructs a transaction wrapper bound to db.
func newTxn(db *DB, raw *rawlmdb.Txn, readonly, holdsWrite bool) *txn {
	return &txn{
		db:         db,
		raw:        raw,
		readonly:   readonly,
		holdsWrite: holdsWrite,
	}
}

// Get returns a copy of the value for key.
func (t *txn) Get(ctx context.Context, key []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateKey(key); err != nil {
		return nil, err
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if err := t.ensureOpenLocked(); err != nil {
		return nil, err
	}

	raw, err := t.raw.Get(t.db.dbi, key)
	if err != nil {
		return nil, mapNotFound(err)
	}
	return cloneBytes(raw), nil
}

// Put stores a key/value pair.
func (t *txn) Put(ctx context.Context, key, value []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if t.readonly {
		return ErrReadOnly
	}
	if err := validateKeyValue(key, value); err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if err := t.ensureOpenLocked(); err != nil {
		return err
	}
	if err := t.raw.Put(t.db.dbi, key, value, 0); err != nil {
		return fmt.Errorf("lmdb: txn put: %w", err)
	}
	return nil
}

// Delete removes key when it exists.
func (t *txn) Delete(ctx context.Context, key []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if t.readonly {
		return ErrReadOnly
	}
	if err := validateKey(key); err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if err := t.ensureOpenLocked(); err != nil {
		return err
	}
	if err := t.raw.Del(t.db.dbi, key, nil); err != nil {
		return mapNotFound(err)
	}
	return nil
}

// Exists reports whether key is present.
func (t *txn) Exists(ctx context.Context, key []byte) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := validateKey(key); err != nil {
		return false, err
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if err := t.ensureOpenLocked(); err != nil {
		return false, err
	}

	_, err := t.raw.Get(t.db.dbi, key)
	if err == nil {
		return true, nil
	}
	if isNotFound(err) {
		return false, nil
	}
	return false, err
}

// Cursor opens a cursor over the named database.
func (t *txn) Cursor(ctx context.Context) (Cursor, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if err := t.ensureOpenLocked(); err != nil {
		return nil, err
	}

	raw, err := t.raw.OpenCursor(t.db.dbi)
	if err != nil {
		return nil, fmt.Errorf("lmdb: open cursor: %w", err)
	}
	return newCursor(t, raw), nil
}

// Commit persists a read-write transaction.
func (t *txn) Commit(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		_ = t.Abort()
		return err
	}
	if t.readonly {
		return t.Abort()
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if err := t.ensureOpenLocked(); err != nil {
		return err
	}

	err := t.raw.Commit()
	t.finishLocked()
	if err != nil {
		return fmt.Errorf("lmdb: commit: %w", err)
	}
	return nil
}

// Abort discards the transaction.
func (t *txn) Abort() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}
	if t.raw != nil {
		t.raw.Abort()
	}
	t.finishLocked()
	return nil
}

// Readonly reports whether the transaction allows writes.
func (t *txn) Readonly() bool {
	return t.readonly
}

// ensureOpenLocked verifies that the transaction is still usable.
func (t *txn) ensureOpenLocked() error {
	if t.closed || t.raw == nil {
		return ErrTxnClosed
	}
	return nil
}

// finishLocked marks the transaction closed and releases engine bookkeeping.
func (t *txn) finishLocked() {
	if t.closed {
		return
	}
	t.closed = true
	t.raw = nil
	t.db.txns.Done()
	if t.holdsWrite {
		t.holdsWrite = false
		runtime.UnlockOSThread()
		t.db.writeMu.Unlock()
	}
}

var _ Txn = (*txn)(nil)
