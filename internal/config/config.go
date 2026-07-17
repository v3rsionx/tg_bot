// Package config loads and validates application configuration from defaults,
// optional JSON, an optional .env file, and environment variable overrides.
//
// Precedence (highest wins):
//  1. process environment variables
//  2. .env file values
//  3. JSON config file values
//  4. package defaults
//
// Loaders are concurrency-safe, support hot reload, never log raw secrets
// (use Config.Redacted / Config.String), and retain the last good config when
// a reload fails validation.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/v3rsi/tgbot-versionx/internal/security"
	"github.com/v3rsi/tgbot-versionx/internal/validator"
)

// Provider is the injectable configuration access contract.
type Provider interface {
	// Get returns a defensive copy of the current validated configuration.
	Get() *Config
	// Reload rebuilds configuration from sources without process restart.
	Reload() (*Config, error)
}

// Config contains all validated application configuration.
type Config struct {
	// Version is the configuration schema version (CONFIG_VERSION).
	Version string

	BotToken         string
	BotOwnerIDs      []int64
	SQLitePath       string
	LMDBIDPath       string
	LMDBPhonePath    string
	LMDBUsernamePath string
	LogLevel         string

	WorkerCount     int
	BatchSize       int
	SearchTimeout   time.Duration
	PointsPerSearch int
	// MaxResults is the preferred field name (MAX_RESULTS).
	MaxResults int
	// MaxSearchResult is retained for existing callers (MAX_SEARCH_RESULT).
	MaxSearchResult int
	CacheTTL        time.Duration
	RateLimit       int
}

// Load reads defaults, optional JSON (configs/config.json), optional .env,
// applies environment-variable overrides, and returns a validated Config.
func Load() (*Config, error) {
	loader := NewLoader(validator.New(), security.New(), Options{
		EnvFile:     defaultEnvFile,
		JSONFile:    defaultJSONFile,
		ApplyDotEnv: true,
	})
	return loader.Load()
}

// MustLoad returns a fully validated configuration or panics. It is intended
// for executable entry points, where invalid configuration must stop startup.
func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}

// Clone returns a deep copy of the configuration.
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}
	out := *c
	if c.BotOwnerIDs != nil {
		out.BotOwnerIDs = append([]int64(nil), c.BotOwnerIDs...)
	}
	return &out
}

// Redacted returns a copy with secrets masked for logging and diagnostics.
func (c *Config) Redacted() *Config {
	out := c.Clone()
	if out == nil {
		return nil
	}
	out.BotToken = maskSecret(out.BotToken)
	return out
}

// String returns a safe diagnostic representation that never includes the bot token.
func (c Config) String() string {
	owners := make([]string, 0, len(c.BotOwnerIDs))
	for _, id := range c.BotOwnerIDs {
		owners = append(owners, fmt.Sprintf("%d", id))
	}
	return fmt.Sprintf(
		"Config{version=%q log=%q owners=[%s] sqlite=%q lmdb_id=%q lmdb_phone=%q lmdb_username=%q workers=%d batch=%d search_timeout=%s points=%d max_results=%d cache_ttl=%s rate_limit=%d token=%s}",
		c.Version,
		c.LogLevel,
		strings.Join(owners, ","),
		c.SQLitePath,
		c.LMDBIDPath,
		c.LMDBPhonePath,
		c.LMDBUsernamePath,
		c.WorkerCount,
		c.BatchSize,
		c.SearchTimeout,
		c.PointsPerSearch,
		c.MaxResults,
		c.CacheTTL,
		c.RateLimit,
		maskSecret(c.BotToken),
	)
}

func maskSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if i := strings.IndexByte(value, ':'); i > 0 && i < len(value)-1 {
		return value[:i] + ":***"
	}
	if len(value) <= 4 {
		return "***"
	}
	return value[:2] + "***"
}

// loadFromEnvironment builds a validated Config from the process environment
// and package defaults. Used by unit tests for env-only coverage.
func loadFromEnvironment() (*Config, error) {
	cfg := Defaults()
	src := &valueSource{
		validator: validator.New(),
		sanitizer: security.New(),
	}
	if err := src.applyOverrides(&cfg); err != nil {
		return nil, err
	}
	alignMaxResults(&cfg)
	if err := validateConfig(&cfg, src.validator, src.sanitizer); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func alignMaxResults(cfg *Config) {
	if cfg.MaxResults > 0 {
		cfg.MaxSearchResult = cfg.MaxResults
		return
	}
	if cfg.MaxSearchResult > 0 {
		cfg.MaxResults = cfg.MaxSearchResult
	}
}

var _ Provider = (*Loader)(nil)
