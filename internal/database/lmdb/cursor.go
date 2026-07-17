package lmdb

import (
	"context"
	"sync"

	rawlmdb "github.com/bmatsuo/lmdb-go/lmdb"
)

// cursor wraps an LMDB cursor and satisfies the Cursor interface.
type cursor struct {
	txn *txn
	raw *rawlmdb.Cursor
	mu  sync.Mutex
}

// newCursor constructs a cursor wrapper bound to txn.
func newCursor(txn *txn, raw *rawlmdb.Cursor) *cursor {
	return &cursor{
		txn: txn,
		raw: raw,
	}
}

// First positions the cursor at the first key/value pair.
func (c *cursor) First(ctx context.Context) (key, value []byte, err error) {
	return c.get(ctx, rawlmdb.First, nil)
}

// Next advances the cursor to the next key/value pair.
func (c *cursor) Next(ctx context.Context) (key, value []byte, err error) {
	return c.get(ctx, rawlmdb.Next, nil)
}

// Seek positions the cursor at the first key greater than or equal to key.
func (c *cursor) Seek(ctx context.Context, key []byte) (foundKey, value []byte, err error) {
	if err := validateKey(key); err != nil {
		return nil, nil, err
	}
	return c.get(ctx, rawlmdb.SetRange, key)
}

// Delete removes the current key/value pair.
func (c *cursor) Delete(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.txn.readonly {
		return ErrReadOnly
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.raw == nil {
		return ErrTxnClosed
	}
	if err := c.raw.Del(0); err != nil {
		return mapNotFound(err)
	}
	return nil
}

// Close releases cursor resources.
func (c *cursor) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.raw == nil {
		return nil
	}
	// Write-txn cursors are also closed by LMDB on commit/abort.
	c.raw.Close()
	c.raw = nil
	return nil
}

// get retrieves a cursor position and returns copied key/value bytes.
func (c *cursor) get(ctx context.Context, op uint, seek []byte) ([]byte, []byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.raw == nil {
		return nil, nil, ErrTxnClosed
	}

	var (
		key []byte
		val []byte
		err error
	)
	if seek == nil {
		key, val, err = c.raw.Get(nil, nil, op)
	} else {
		key, val, err = c.raw.Get(seek, nil, op)
	}
	if err != nil {
		return nil, nil, mapNotFound(err)
	}
	return cloneBytes(key), cloneBytes(val), nil
}

var _ Cursor = (*cursor)(nil)
