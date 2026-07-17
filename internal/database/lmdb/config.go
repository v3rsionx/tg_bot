package lmdb

import (
	"fmt"
	"path/filepath"
	"time"
)

const (
	defaultInitialMapSize int64 = 1 << 30 // 1 GiB
	defaultMaxMapSize     int64 = 1 << 40 // 1 TiB
	defaultMapGrowth      int64 = 1 << 30 // grow by 1 GiB
	defaultMaxReaders           = 1024
	defaultMaxDBs               = 1
	defaultDBName               = "data"
	defaultFileMode             = 0o644
)

// Config controls LMDB environment creation and runtime growth behavior.
type Config struct {
	// Path is the directory that stores the LMDB environment files.
	Path string
	// DBName is the named sub-database opened inside the environment.
	DBName string
	// InitialMapSize is the starting memory map size in bytes.
	InitialMapSize int64
	// MaxMapSize is the upper bound for automatic map growth in bytes.
	MaxMapSize int64
	// MapGrowth is the number of bytes added when the map is full.
	MapGrowth int64
	// MaxReaders is the maximum number of simultaneous read transactions.
	MaxReaders int
	// MaxDBs is the maximum number of named databases in the environment.
	MaxDBs int
	// FileMode is the permission mode used when creating environment files.
	FileMode uint
	// ReadOnly opens the environment without write capability.
	ReadOnly bool
	// NoSync disables flush-on-commit for higher write throughput.
	NoSync bool
	// NoMetaSync disables metadata flush-on-commit.
	NoMetaSync bool
	// WriteMap enables a writable memory map when supported by the platform.
	WriteMap bool
	// OpenTimeout bounds how long Open waits for the environment lock.
	OpenTimeout time.Duration
}

// Validate checks that required configuration values are present and sane.
func (c Config) Validate() error {
	if c.Path == "" {
		return fmt.Errorf("lmdb: Path is required")
	}
	if filepath.Clean(c.Path) == "." {
		return fmt.Errorf("lmdb: Path must identify a directory")
	}
	if c.InitialMapSize < 0 {
		return fmt.Errorf("lmdb: InitialMapSize must be >= 0")
	}
	if c.MaxMapSize < 0 {
		return fmt.Errorf("lmdb: MaxMapSize must be >= 0")
	}
	if c.MapGrowth < 0 {
		return fmt.Errorf("lmdb: MapGrowth must be >= 0")
	}
	if c.MaxReaders < 0 {
		return fmt.Errorf("lmdb: MaxReaders must be >= 0")
	}
	if c.MaxDBs < 0 {
		return fmt.Errorf("lmdb: MaxDBs must be >= 0")
	}
	cfg := c.withDefaults()
	if cfg.MaxMapSize < cfg.InitialMapSize {
		return fmt.Errorf("lmdb: MaxMapSize must be >= InitialMapSize")
	}
	if cfg.MapGrowth == 0 {
		return fmt.Errorf("lmdb: MapGrowth must be > 0")
	}
	return nil
}

// withDefaults returns a copy of Config with production-oriented defaults applied.
func (c Config) withDefaults() Config {
	if c.DBName == "" {
		c.DBName = defaultDBName
	}
	if c.InitialMapSize == 0 {
		c.InitialMapSize = defaultInitialMapSize
	}
	if c.MaxMapSize == 0 {
		c.MaxMapSize = defaultMaxMapSize
	}
	if c.MapGrowth == 0 {
		c.MapGrowth = defaultMapGrowth
	}
	if c.MaxReaders == 0 {
		c.MaxReaders = defaultMaxReaders
	}
	if c.MaxDBs == 0 {
		c.MaxDBs = defaultMaxDBs
	}
	if c.FileMode == 0 {
		c.FileMode = defaultFileMode
	}
	return c
}
