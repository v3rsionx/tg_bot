package config

import "testing"

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
}
