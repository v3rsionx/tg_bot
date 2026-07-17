package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/v3rsionx/tg_bot/internal/security"
	"github.com/v3rsionx/tg_bot/internal/validator"
)

func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("BOT_TOKEN", "123456789:abcdefghijklmnopqrstuvwxyz_123456789")
	t.Setenv("BOT_OWNER_IDS", "12345,67890")
	t.Setenv("SQLITE_PATH", "./data/bot.db")
	t.Setenv("LMDB_ID_PATH", "./data/lmdb/id")
	t.Setenv("LMDB_PHONE_PATH", "./data/lmdb/phone")
	t.Setenv("LMDB_USERNAME_PATH", "./data/lmdb/username")
	t.Setenv("LOG_LEVEL", "INFO")
	t.Setenv("POINTS_PER_SEARCH", "1")
	t.Setenv("MAX_SEARCH_RESULT", "100")

	config, err := loadFromEnvironment()
	if err != nil {
		t.Fatalf("loadFromEnvironment() error = %v", err)
	}

	if config.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", config.LogLevel, "info")
	}
	if len(config.BotOwnerIDs) != 2 {
		t.Fatalf("BotOwnerIDs length = %d, want 2", len(config.BotOwnerIDs))
	}
	if config.Version != CurrentConfigVersion {
		t.Errorf("Version = %q, want %q", config.Version, CurrentConfigVersion)
	}
	if config.WorkerCount != 4 {
		t.Errorf("WorkerCount = %d, want 4", config.WorkerCount)
	}
	if config.MaxResults != 100 || config.MaxSearchResult != 100 {
		t.Errorf("MaxResults/MaxSearchResult = %d/%d, want 100/100", config.MaxResults, config.MaxSearchResult)
	}
	if config.SearchTimeout != 5*time.Second {
		t.Errorf("SearchTimeout = %v, want 5s", config.SearchTimeout)
	}
}

func TestLoadFromEnvironmentRejectsInvalidValues(t *testing.T) {
	t.Setenv("BOT_TOKEN", "invalid")
	t.Setenv("BOT_OWNER_IDS", "12345,12345")
	t.Setenv("SQLITE_PATH", "./data/bot.db")
	t.Setenv("LMDB_ID_PATH", "./data/lmdb/shared")
	t.Setenv("LMDB_PHONE_PATH", "./data/lmdb/shared")
	t.Setenv("LMDB_USERNAME_PATH", "./data/lmdb/username")
	t.Setenv("LOG_LEVEL", "verbose")
	t.Setenv("POINTS_PER_SEARCH", "0")
	t.Setenv("MAX_SEARCH_RESULT", "100")

	if _, err := loadFromEnvironment(); err == nil {
		t.Fatal("loadFromEnvironment() error = nil, want validation error")
	}
}

func TestParseEnvLine(t *testing.T) {
	key, value, ok, err := parseEnvLine(`export LOG_LEVEL="info"`)
	if err != nil {
		t.Fatalf("parseEnvLine() error = %v", err)
	}
	if !ok || key != "LOG_LEVEL" || value != "info" {
		t.Fatalf("parseEnvLine() = (%q, %q, %t), want (%q, %q, true)", key, value, ok, "LOG_LEVEL", "info")
	}
	if _, _, _, err := parseEnvLine(`bad-key=1`); err == nil {
		t.Fatal("parseEnvLine() error = nil, want invalid key error")
	}
}

func TestConfigRedaction(t *testing.T) {
	cfg := &Config{
		Version:  CurrentConfigVersion,
		BotToken: "123456789:abcdefghijklmnopqrstuvwxyz_123456789",
		LogLevel: "info",
	}
	redacted := cfg.Redacted()
	if strings.Contains(redacted.BotToken, "abcdefgh") {
		t.Fatalf("Redacted token still contains secret: %q", redacted.BotToken)
	}
	if strings.Contains(cfg.String(), "abcdefgh") {
		t.Fatalf("String() leaked token secret: %s", cfg.String())
	}
	if cfg.BotToken == redacted.BotToken {
		t.Fatal("Redacted() mutated or aliased original token unexpectedly")
	}
}

func TestLoaderJSONAndEnvOverride(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "config.json")
	envPath := filepath.Join(dir, ".env")

	jsonBody := `{
  "version": "1",
  "bot_token": "123456789:abcdefghijklmnopqrstuvwxyz_123456789",
  "bot_owner_ids": [1],
  "sqlite_path": "./data/bot.db",
  "lmdb_id_path": "./data/lmdb/id",
  "lmdb_phone_path": "./data/lmdb/phone",
  "lmdb_username_path": "./data/lmdb/username",
  "log_level": "warn",
  "worker_count": 8,
  "batch_size": 500,
  "search_timeout": "3s",
  "points_per_search": 2,
  "max_results": 50,
  "cache_ttl": "1m",
  "rate_limit": 10
}`
	if err := os.WriteFile(jsonPath, []byte(jsonBody), 0o600); err != nil {
		t.Fatalf("WriteFile JSON: %v", err)
	}
	if err := os.WriteFile(envPath, []byte("LOG_LEVEL=error\nWORKER_COUNT=16\n"), 0o600); err != nil {
		t.Fatalf("WriteFile env: %v", err)
	}

	t.Setenv("BOT_TOKEN", "123456789:abcdefghijklmnopqrstuvwxyz_123456789")
	t.Setenv("BOT_OWNER_IDS", "1")
	t.Setenv("SQLITE_PATH", "./data/bot.db")
	t.Setenv("LMDB_ID_PATH", "./data/lmdb/id")
	t.Setenv("LMDB_PHONE_PATH", "./data/lmdb/phone")
	t.Setenv("LMDB_USERNAME_PATH", "./data/lmdb/username")
	_ = os.Unsetenv("LOG_LEVEL")
	_ = os.Unsetenv("WORKER_COUNT")

	loader := NewLoader(validator.New(), security.New(), Options{
		EnvFile:     envPath,
		JSONFile:    jsonPath,
		ApplyDotEnv: false,
	})
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.LogLevel != "error" {
		t.Errorf("LogLevel = %q, want error (.env override)", cfg.LogLevel)
	}
	if cfg.WorkerCount != 16 {
		t.Errorf("WorkerCount = %d, want 16 (.env override)", cfg.WorkerCount)
	}
	if cfg.MaxResults != 50 {
		t.Errorf("MaxResults = %d, want 50", cfg.MaxResults)
	}

	var reloadErr error
	loader.OnReloadError(func(err error) { reloadErr = err })

	t.Setenv("WORKER_COUNT", "32")
	cfg, err = loader.Reload()
	if err != nil {
		t.Fatalf("Reload() error = %v", err)
	}
	if cfg.WorkerCount != 32 {
		t.Errorf("WorkerCount = %d, want 32 (env override)", cfg.WorkerCount)
	}

	t.Setenv("BOT_TOKEN", "invalid")
	if _, err := loader.Reload(); err == nil {
		t.Fatal("Reload() error = nil, want validation error")
	}
	if reloadErr == nil {
		t.Fatal("OnReloadError was not invoked")
	}
	if got := loader.Get(); got == nil || got.WorkerCount != 32 {
		t.Fatalf("Get() after failed reload = %#v, want previous good config", got)
	}
}

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.Version != CurrentConfigVersion {
		t.Errorf("Version = %q, want %q", cfg.Version, CurrentConfigVersion)
	}
	if cfg.RateLimit != 5 {
		t.Errorf("RateLimit = %d, want 5", cfg.RateLimit)
	}
}
