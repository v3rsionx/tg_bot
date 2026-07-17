package lmdb

import (
	"context"
	"fmt"

	rawlmdb "github.com/bmatsuo/lmdb-go/lmdb"
)

// withRead executes fn inside a read-only transaction.
func (db *DB) withRead(ctx context.Context, fn func(txn *rawlmdb.Txn) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	db.mu.RLock()
	if err := db.ensureOpenLocked(); err != nil {
		db.mu.RUnlock()
		return err
	}
	env := db.env
	db.txns.Add(1)
	db.mu.RUnlock()
	defer db.txns.Done()

	return env.View(func(txn *rawlmdb.Txn) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return fn(txn)
	})
}

// withWrite executes fn inside a read-write transaction and retries after map growth.
func (db *DB) withWrite(ctx context.Context, fn func(txn *rawlmdb.Txn) error) error {
	if db.cfg.ReadOnly {
		return ErrReadOnly
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := db.runWrite(ctx, fn)
		if err == nil {
			return nil
		}
		if !isMapFull(err) {
			return err
		}

		db.writeMu.Lock()
		growErr := db.growMapLocked()
		db.writeMu.Unlock()
		if growErr != nil {
			return growErr
		}
	}
}

// runWrite executes one write transaction attempt under the write mutex.
func (db *DB) runWrite(ctx context.Context, fn func(txn *rawlmdb.Txn) error) error {
	db.writeMu.Lock()
	defer db.writeMu.Unlock()

	db.mu.RLock()
	if err := db.ensureOpenLocked(); err != nil {
		db.mu.RUnlock()
		return err
	}
	env := db.env
	db.txns.Add(1)
	db.mu.RUnlock()
	defer db.txns.Done()

	return env.Update(func(txn *rawlmdb.Txn) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return fn(txn)
	})
}

// growMapLocked increases the LMDB map size. Caller must hold writeMu.
func (db *DB) growMapLocked() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if err := db.ensureOpenLocked(); err != nil {
		return err
	}

	next := db.mapSize + db.cfg.MapGrowth
	if next > db.cfg.MaxMapSize {
		if db.mapSize >= db.cfg.MaxMapSize {
			return ErrMapSizeLimit
		}
		next = db.cfg.MaxMapSize
	}
	if next <= db.mapSize {
		return ErrMapSizeLimit
	}

	if err := db.env.SetMapSize(next); err != nil {
		return fmt.Errorf("lmdb: grow map size to %d: %w", next, err)
	}
	db.mapSize = next
	return nil
}
