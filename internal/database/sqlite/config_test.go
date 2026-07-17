package sqlite

import (
	"testing"
	"time"
)

// TestConfigValidateRejectsEmptyPath ensures Path is required.
func TestConfigValidateRejectsEmptyPath(t *testing.T) {
	cfg := Config{}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want error")
	}
}

// TestConfigWithDefaultsAppliesProductionDefaults verifies default values.
func TestConfigWithDefaultsAppliesProductionDefaults(t *testing.T) {
	cfg := Config{Path: "data/bot.db"}.withDefaults()

	if cfg.MigrationsPath != defaultMigrationsDir {
		t.Fatalf("MigrationsPath = %q, want %q", cfg.MigrationsPath, defaultMigrationsDir)
	}
	if cfg.MaxOpenConns != defaultMaxOpenConns {
		t.Fatalf("MaxOpenConns = %d, want %d", cfg.MaxOpenConns, defaultMaxOpenConns)
	}
	if cfg.BusyTimeout != time.Duration(defaultBusyTimeoutMS)*time.Millisecond {
		t.Fatalf("BusyTimeout = %s, want %dms", cfg.BusyTimeout, defaultBusyTimeoutMS)
	}
}

// TestParseMigrationVersionExtractsLeadingInteger verifies migration naming rules.
func TestParseMigrationVersionExtractsLeadingInteger(t *testing.T) {
	version, err := parseMigrationVersion("migrations/000001_init.up.sql")
	if err != nil {
		t.Fatalf("parseMigrationVersion() error = %v", err)
	}
	if version != 1 {
		t.Fatalf("version = %d, want 1", version)
	}
}

// TestFormatAndParseTimeRoundTrip ensures timestamp encoding is stable.
func TestFormatAndParseTimeRoundTrip(t *testing.T) {
	original := time.Date(2026, 7, 16, 12, 30, 45, 123456789, time.UTC)
	encoded := formatTime(original)

	parsed, err := parseTime(encoded)
	if err != nil {
		t.Fatalf("parseTime() error = %v", err)
	}
	if !parsed.Equal(original) {
		t.Fatalf("parsed = %s, want %s", parsed, original)
	}
}
