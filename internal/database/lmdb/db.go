package lmdb

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"

	rawlmdb "github.com/bmatsuo/lmdb-go/lmdb"
)

// DB is a thread-safe LMDB storage engine.
type DB struct {
	mu      sync.RWMutex
	writeMu sync.Mutex
	txns    sync.WaitGroup
	cfg     Config
	env     *rawlmdb.Env
	dbi     rawlmdb.DBI
	mapSize int64
	opened  bool
	closed  bool
}

// New constructs a closed DB from configuration. Call Open before use.
func New(cfg Config) (*DB, error) {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &DB{
		cfg:     cfg,
		mapSize: cfg.InitialMapSize,
	}, nil
}

// Open initializes the LMDB environment and named database.
func (db *DB) Open(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return ErrClosed
	}
	if db.opened {
		return nil
	}

	if err := os.MkdirAll(db.cfg.Path, 0o755); err != nil {
		return fmt.Errorf("lmdb: create path %q: %w", db.cfg.Path, err)
	}

	env, err := rawlmdb.NewEnv()
	if err != nil {
		return fmt.Errorf("lmdb: create env: %w", err)
	}

	if err := env.SetMapSize(db.mapSize); err != nil {
		_ = env.Close()
		return fmt.Errorf("lmdb: set map size: %w", err)
	}
	if err := env.SetMaxReaders(db.cfg.MaxReaders); err != nil {
		_ = env.Close()
		return fmt.Errorf("lmdb: set max readers: %w", err)
	}
	if err := env.SetMaxDBs(db.cfg.MaxDBs); err != nil {
		_ = env.Close()
		return fmt.Errorf("lmdb: set max dbs: %w", err)
	}

	flags := uint(0)
	if db.cfg.ReadOnly {
		flags |= rawlmdb.Readonly
	}
	if db.cfg.NoSync {
		flags |= rawlmdb.NoSync
	}
	if db.cfg.NoMetaSync {
		flags |= rawlmdb.NoMetaSync
	}
	if db.cfg.WriteMap {
		flags |= rawlmdb.WriteMap
	}

	if err := env.Open(db.cfg.Path, flags, os.FileMode(db.cfg.FileMode)); err != nil {
		_ = env.Close()
		return fmt.Errorf("lmdb: open env %q: %w", db.cfg.Path, err)
	}

	dbi, err := openDBI(env, db.cfg.DBName, db.cfg.ReadOnly)
	if err != nil {
		_ = env.Close()
		return err
	}

	if err := ctx.Err(); err != nil {
		_ = env.Close()
		return err
	}

	db.env = env
	db.dbi = dbi
	db.opened = true
	return nil
}

// openDBI opens or creates the named database inside env.
func openDBI(env *rawlmdb.Env, name string, readOnly bool) (rawlmdb.DBI, error) {
	var dbi rawlmdb.DBI
	if readOnly {
		err := env.View(func(txn *rawlmdb.Txn) error {
			opened, err := txn.OpenDBI(name, 0)
			if err != nil {
				return err
			}
			dbi = opened
			return nil
		})
		if err != nil {
			return 0, fmt.Errorf("lmdb: open database %q: %w", name, err)
		}
		return dbi, nil
	}

	err := env.Update(func(txn *rawlmdb.Txn) error {
		opened, err := txn.OpenDBI(name, rawlmdb.Create)
		if err != nil {
			return err
		}
		dbi = opened
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("lmdb: open database %q: %w", name, err)
	}
	return dbi, nil
}

// OpenDB constructs, opens, and returns a ready LMDB engine.
func OpenDB(ctx context.Context, cfg Config) (*DB, error) {
	db, err := New(cfg)
	if err != nil {
		return nil, err
	}
	if err := db.Open(ctx); err != nil {
		return nil, err
	}
	return db, nil
}

// Close releases environment resources.
func (db *DB) Close() error {
	db.mu.Lock()
	if db.closed {
		db.mu.Unlock()
		return nil
	}
	db.closed = true
	db.opened = false
	env := db.env
	db.env = nil
	db.mu.Unlock()

	db.txns.Wait()

	if env == nil {
		return nil
	}
	if err := env.Close(); err != nil {
		return fmt.Errorf("lmdb: close: %w", err)
	}
	return nil
}

// Put stores a single key/value pair.
func (db *DB) Put(ctx context.Context, key, value []byte) error {
	if err := validateKeyValue(key, value); err != nil {
		return err
	}
	return db.withWrite(ctx, func(txn *rawlmdb.Txn) error {
		return txn.Put(db.dbi, key, value, 0)
	})
}

// Get returns a copy of the value for key.
func (db *DB) Get(ctx context.Context, key []byte) ([]byte, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	var value []byte
	err := db.withRead(ctx, func(txn *rawlmdb.Txn) error {
		raw, err := txn.Get(db.dbi, key)
		if err != nil {
			return err
		}
		value = cloneBytes(raw)
		return nil
	})
	if err != nil {
		return nil, mapNotFound(err)
	}
	return value, nil
}

// Delete removes key when it exists.
func (db *DB) Delete(ctx context.Context, key []byte) error {
	if err := validateKey(key); err != nil {
		return err
	}
	err := db.withWrite(ctx, func(txn *rawlmdb.Txn) error {
		return txn.Del(db.dbi, key, nil)
	})
	return mapNotFound(err)
}

// Exists reports whether key is present.
func (db *DB) Exists(ctx context.Context, key []byte) (bool, error) {
	if err := validateKey(key); err != nil {
		return false, err
	}

	err := db.withRead(ctx, func(txn *rawlmdb.Txn) error {
		_, err := txn.Get(db.dbi, key)
		return err
	})
	if err == nil {
		return true, nil
	}
	if isNotFound(err) {
		return false, nil
	}
	return false, err
}

// BatchPut stores many key/value pairs in one write transaction.
func (db *DB) BatchPut(ctx context.Context, items []KeyValue) error {
	if len(items) == 0 {
		return nil
	}
	for i := range items {
		if err := validateKeyValue(items[i].Key, items[i].Value); err != nil {
			return fmt.Errorf("lmdb: batch item %d: %w", i, err)
		}
	}

	return db.withWrite(ctx, func(txn *rawlmdb.Txn) error {
		for i := range items {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := txn.Put(db.dbi, items[i].Key, items[i].Value, 0); err != nil {
				return fmt.Errorf("lmdb: batch put item %d: %w", i, err)
			}
		}
		return nil
	})
}

// BatchDelete removes many keys in one write transaction.
func (db *DB) BatchDelete(ctx context.Context, keys [][]byte) error {
	if len(keys) == 0 {
		return nil
	}
	for i := range keys {
		if err := validateKey(keys[i]); err != nil {
			return fmt.Errorf("lmdb: batch delete key %d: %w", i, err)
		}
	}

	return db.withWrite(ctx, func(txn *rawlmdb.Txn) error {
		for i := range keys {
			if err := ctx.Err(); err != nil {
				return err
			}
			err := txn.Del(db.dbi, keys[i], nil)
			if err != nil && !isNotFound(err) {
				return fmt.Errorf("lmdb: batch delete key %d: %w", i, err)
			}
		}
		return nil
	})
}

// Stats returns environment and database statistics.
func (db *DB) Stats(ctx context.Context) (Stats, error) {
	if err := ctx.Err(); err != nil {
		return Stats{}, err
	}

	db.mu.RLock()
	if err := db.ensureOpenLocked(); err != nil {
		db.mu.RUnlock()
		return Stats{}, err
	}
	env := db.env
	dbi := db.dbi
	db.txns.Add(1)
	db.mu.RUnlock()
	defer db.txns.Done()

	info, err := env.Info()
	if err != nil {
		return Stats{}, fmt.Errorf("lmdb: env info: %w", err)
	}
	envStat, err := env.Stat()
	if err != nil {
		return Stats{}, fmt.Errorf("lmdb: env stat: %w", err)
	}

	var dbStat *rawlmdb.Stat
	err = env.View(func(txn *rawlmdb.Txn) error {
		stat, statErr := txn.Stat(dbi)
		if statErr != nil {
			return statErr
		}
		dbStat = stat
		return nil
	})
	if err != nil {
		return Stats{}, fmt.Errorf("lmdb: db stat: %w", err)
	}

	return Stats{
		MapSize:       info.MapSize,
		LastPageNo:    info.LastPNO,
		LastTxnID:     info.LastTxnID,
		MaxReaders:    info.MaxReaders,
		NumReaders:    info.NumReaders,
		Entries:       dbStat.Entries,
		Depth:         dbStat.Depth,
		BranchPages:   dbStat.BranchPages,
		LeafPages:     dbStat.LeafPages,
		OverflowPages: dbStat.OverflowPages,
		PageSize:      envStat.PSize,
	}, nil
}

// Sync flushes data to durable storage.
func (db *DB) Sync(ctx context.Context, force bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	db.mu.RLock()
	if err := db.ensureOpenLocked(); err != nil {
		db.mu.RUnlock()
		return err
	}
	env := db.env
	db.mu.RUnlock()

	if err := env.Sync(force); err != nil {
		return fmt.Errorf("lmdb: sync: %w", err)
	}
	return nil
}

// Reader begins a read-only transaction.
func (db *DB) Reader(ctx context.Context) (Txn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	db.mu.RLock()
	if err := db.ensureOpenLocked(); err != nil {
		db.mu.RUnlock()
		return nil, err
	}
	env := db.env
	db.txns.Add(1)
	db.mu.RUnlock()

	raw, err := env.BeginTxn(nil, rawlmdb.Readonly)
	if err != nil {
		db.txns.Done()
		return nil, fmt.Errorf("lmdb: begin read transaction: %w", err)
	}
	return newTxn(db, raw, true, false), nil
}

// Writer begins a read-write transaction.
// The calling goroutine is locked to its OS thread until Commit or Abort.
func (db *DB) Writer(ctx context.Context) (Txn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if db.cfg.ReadOnly {
		return nil, ErrReadOnly
	}

	db.writeMu.Lock()
	runtime.LockOSThread()

	db.mu.RLock()
	if err := db.ensureOpenLocked(); err != nil {
		db.mu.RUnlock()
		runtime.UnlockOSThread()
		db.writeMu.Unlock()
		return nil, err
	}
	env := db.env
	db.txns.Add(1)
	db.mu.RUnlock()

	raw, err := env.BeginTxn(nil, 0)
	if err != nil {
		db.txns.Done()
		runtime.UnlockOSThread()
		db.writeMu.Unlock()
		return nil, fmt.Errorf("lmdb: begin write transaction: %w", err)
	}
	return newTxn(db, raw, false, true), nil
}

// ensureOpenLocked verifies that the environment is open. Caller must hold a lock.
func (db *DB) ensureOpenLocked() error {
	if db.closed || !db.opened || db.env == nil {
		return ErrClosed
	}
	return nil
}

var _ Engine = (*DB)(nil)
