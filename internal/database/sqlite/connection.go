package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// ConnectionManager owns a thread-safe *sql.DB pool for SQLite.
type ConnectionManager struct {
	mu  sync.RWMutex
	db  *sql.DB
	cfg Config
}

// NewConnectionManager opens and configures a SQLite connection pool.
func NewConnectionManager(cfg Config) (*ConnectionManager, error) {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if err := ensureParentDir(cfg.Path); err != nil {
		return nil, err
	}

	dsn := buildDSN(cfg)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	manager := &ConnectionManager{
		db:  db,
		cfg: cfg,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := manager.Ping(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := manager.applyPragmas(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return manager, nil
}

// DB returns the underlying thread-safe database handle.
func (m *ConnectionManager) DB() *sql.DB {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.db
}

// Ping verifies that the database connection is usable.
func (m *ConnectionManager) Ping(ctx context.Context) error {
	db := m.DB()
	if db == nil {
		return fmt.Errorf("sqlite: connection is closed")
	}
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("sqlite: ping: %w", err)
	}
	return nil
}

// Close closes the underlying connection pool.
func (m *ConnectionManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.db == nil {
		return nil
	}
	err := m.db.Close()
	m.db = nil
	if err != nil {
		return fmt.Errorf("sqlite: close: %w", err)
	}
	return nil
}

// applyPragmas configures production SQLite runtime settings.
func (m *ConnectionManager) applyPragmas(ctx context.Context) error {
	pragmas := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA temp_store = MEMORY;",
		fmt.Sprintf("PRAGMA busy_timeout = %d;", int(m.cfg.BusyTimeout.Milliseconds())),
	}

	for _, pragma := range pragmas {
		if _, err := m.db.ExecContext(ctx, pragma); err != nil {
			return fmt.Errorf("sqlite: apply %q: %w", pragma, err)
		}
	}
	return nil
}

// buildDSN constructs a SQLite DSN with safe defaults for concurrent access.
func buildDSN(cfg Config) string {
	busyTimeout := int(cfg.BusyTimeout.Milliseconds())
	options := fmt.Sprintf(
		"_foreign_keys=1&_busy_timeout=%d&_journal_mode=WAL&_txlock=immediate",
		busyTimeout,
	)

	path := strings.TrimSpace(cfg.Path)
	switch {
	case path == ":memory:":
		return "file:memdb?mode=memory&cache=shared&" + options
	case strings.HasPrefix(path, "file:"):
		if strings.Contains(path, "?") {
			return path + "&" + options
		}
		return path + "?" + options
	default:
		return fmt.Sprintf("file:%s?%s", filepath.ToSlash(path), options)
	}
}

// ensureParentDir creates the parent directory for the database file when needed.
func ensureParentDir(path string) error {
	path = strings.TrimSpace(path)
	if path == "" || path == ":memory:" || strings.HasPrefix(path, "file:") {
		return nil
	}

	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("sqlite: create database directory %q: %w", dir, err)
	}
	return nil
}
