package sqlite

import (
	"fmt"
	"path/filepath"
	"time"
)

const (
	defaultMaxOpenConns  = 1
	defaultMaxIdleConns  = 1
	defaultBusyTimeoutMS = 5000
	defaultMigrationsDir = "migrations"
)

// Config controls SQLite connection, pool, and migration behavior.
type Config struct {
	// Path is the SQLite database file path (for example data/bot.db).
	Path string
	// MigrationsPath is the directory containing numbered SQL migration files.
	MigrationsPath string
	// MaxOpenConns limits concurrently open SQLite connections.
	MaxOpenConns int
	// MaxIdleConns limits idle connections retained in the pool.
	MaxIdleConns int
	// BusyTimeout is how long SQLite waits when the database is locked.
	BusyTimeout time.Duration
	// ConnMaxLifetime optionally expires pooled connections. Zero disables expiry.
	ConnMaxLifetime time.Duration
	// ConnMaxIdleTime optionally expires idle pooled connections. Zero disables expiry.
	ConnMaxIdleTime time.Duration
}

// Validate checks that required configuration values are present and sane.
func (c Config) Validate() error {
	if c.Path == "" {
		return fmt.Errorf("sqlite: Path is required")
	}
	if filepath.Clean(c.Path) == "." {
		return fmt.Errorf("sqlite: Path must identify a database file")
	}
	if c.MaxOpenConns < 0 {
		return fmt.Errorf("sqlite: MaxOpenConns must be >= 0")
	}
	if c.MaxIdleConns < 0 {
		return fmt.Errorf("sqlite: MaxIdleConns must be >= 0")
	}
	if c.BusyTimeout < 0 {
		return fmt.Errorf("sqlite: BusyTimeout must be >= 0")
	}
	return nil
}

// withDefaults returns a copy of Config with production-safe defaults applied.
func (c Config) withDefaults() Config {
	if c.MigrationsPath == "" {
		c.MigrationsPath = defaultMigrationsDir
	}
	if c.MaxOpenConns == 0 {
		c.MaxOpenConns = defaultMaxOpenConns
	}
	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = defaultMaxIdleConns
	}
	if c.BusyTimeout == 0 {
		c.BusyTimeout = time.Duration(defaultBusyTimeoutMS) * time.Millisecond
	}
	return c
}
