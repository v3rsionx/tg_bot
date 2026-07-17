package search

import (
	"fmt"
	"time"
)

const (
	defaultTimeout      = 2 * time.Second
	defaultCacheTTL     = 5 * time.Minute
	defaultCacheMaxSize = 100_000
)

// Config controls search timeouts and cache behavior.
type Config struct {
	// Timeout bounds each lookup when the caller context has no deadline.
	Timeout time.Duration
	// DisableCache turns off the in-memory exact-lookup cache.
	// Cache is enabled by default.
	DisableCache bool
	// CacheTTL is the maximum age of a cached hit before eviction.
	CacheTTL time.Duration
	// CacheMaxSize is the maximum number of cached entries.
	CacheMaxSize int
}

// Validate checks search configuration values.
func (c Config) Validate() error {
	if c.Timeout < 0 {
		return fmt.Errorf("search: Timeout must be >= 0")
	}
	if c.CacheTTL < 0 {
		return fmt.Errorf("search: CacheTTL must be >= 0")
	}
	if c.CacheMaxSize < 0 {
		return fmt.Errorf("search: CacheMaxSize must be >= 0")
	}
	return nil
}

// withDefaults returns a copy of Config with production defaults applied.
func (c Config) withDefaults() Config {
	if c.Timeout == 0 {
		c.Timeout = defaultTimeout
	}
	if c.CacheTTL == 0 {
		c.CacheTTL = defaultCacheTTL
	}
	if c.CacheMaxSize == 0 {
		c.CacheMaxSize = defaultCacheMaxSize
	}
	return c
}

// cacheEnabled reports whether caching should be active.
func (c Config) cacheEnabled() bool {
	return !c.DisableCache
}
