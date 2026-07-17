package lmdb

import "testing"

// TestConfigValidateRejectsEmptyPath ensures Path is required.
func TestConfigValidateRejectsEmptyPath(t *testing.T) {
	if err := (Config{}).Validate(); err == nil {
		t.Fatal("Validate() error = nil, want error")
	}
}

// TestConfigWithDefaultsAppliesLargeMapDefaults verifies scale-oriented defaults.
func TestConfigWithDefaultsAppliesLargeMapDefaults(t *testing.T) {
	cfg := Config{Path: "data/lmdb/id"}.withDefaults()

	if cfg.InitialMapSize != defaultInitialMapSize {
		t.Fatalf("InitialMapSize = %d, want %d", cfg.InitialMapSize, defaultInitialMapSize)
	}
	if cfg.MaxReaders != defaultMaxReaders {
		t.Fatalf("MaxReaders = %d, want %d", cfg.MaxReaders, defaultMaxReaders)
	}
	if cfg.DBName != defaultDBName {
		t.Fatalf("DBName = %q, want %q", cfg.DBName, defaultDBName)
	}
}

// TestConfigValidateRejectsMaxBelowInitial ensures map bounds are coherent.
func TestConfigValidateRejectsMaxBelowInitial(t *testing.T) {
	cfg := Config{
		Path:           "data/lmdb/id",
		InitialMapSize: 1 << 20,
		MaxMapSize:     1 << 10,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want error")
	}
}
