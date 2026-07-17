package validator

import (
	"testing"
	"time"
)

func TestTelegramToken(t *testing.T) {
	v := New()
	if err := v.TelegramToken("123456789:abcdefghijklmnopqrstuvwxyz_123456789"); err != nil {
		t.Fatalf("TelegramToken() error = %v", err)
	}
	if err := v.TelegramToken("invalid"); err == nil {
		t.Fatal("TelegramToken() error = nil, want validation error")
	}
}

func TestOwnerIDsAndCSV(t *testing.T) {
	v := New()
	ids, err := v.OwnerIDsCSV("1,2,3")
	if err != nil {
		t.Fatalf("OwnerIDsCSV() error = %v", err)
	}
	if len(ids) != 3 {
		t.Fatalf("OwnerIDsCSV() len = %d, want 3", len(ids))
	}
	if err := v.OwnerIDs([]int64{1, 1}); err == nil {
		t.Fatal("OwnerIDs() error = nil, want duplicate error")
	}
}

func TestPathsWorkersBatchTimeout(t *testing.T) {
	v := New()
	if err := v.SQLitePath("./data/bot.db"); err != nil {
		t.Fatalf("SQLitePath() error = %v", err)
	}
	if err := v.LMDBPath("LMDB_ID_PATH", "./data/lmdb/id"); err != nil {
		t.Fatalf("LMDBPath() error = %v", err)
	}
	if err := v.DistinctPaths(map[string]string{
		"LMDB_ID_PATH":    "./data/lmdb/id",
		"LMDB_PHONE_PATH": "./data/lmdb/phone",
	}); err != nil {
		t.Fatalf("DistinctPaths() error = %v", err)
	}
	if err := v.SQLitePath("../secret.db"); err == nil {
		t.Fatal("SQLitePath() error = nil, want traversal error")
	}
	if err := v.WorkerCount(4); err != nil {
		t.Fatalf("WorkerCount() error = %v", err)
	}
	if err := v.BatchSize(1000); err != nil {
		t.Fatalf("BatchSize() error = %v", err)
	}
	if err := v.Timeout("SEARCH_TIMEOUT", 5*time.Second); err != nil {
		t.Fatalf("Timeout() error = %v", err)
	}
	if err := v.MaxResults(100); err != nil {
		t.Fatalf("MaxResults() error = %v", err)
	}
	if err := v.RateLimit(5); err != nil {
		t.Fatalf("RateLimit() error = %v", err)
	}
	if err := v.WorkerCount(0); err == nil {
		t.Fatal("WorkerCount(0) error = nil, want validation error")
	}
	if err := v.MaxResults(0); err == nil {
		t.Fatal("MaxResults(0) error = nil, want validation error")
	}
}

func TestPhoneUsernameUserIDSearchIDCommand(t *testing.T) {
	v := New()
	if err := v.Phone("+1 (555) 123-4567"); err != nil {
		t.Fatalf("Phone() error = %v", err)
	}
	if err := v.Username("@Valid_User1"); err != nil {
		t.Fatalf("Username() error = %v", err)
	}
	if err := v.TelegramUserID(42); err != nil {
		t.Fatalf("TelegramUserID() error = %v", err)
	}
	if err := v.SearchID("123456789"); err != nil {
		t.Fatalf("SearchID() error = %v", err)
	}
	if err := v.Command("/start@mybot"); err != nil {
		t.Fatalf("Command() error = %v", err)
	}
	if err := v.Phone("12"); err == nil {
		t.Fatal("Phone() error = nil, want validation error")
	}
	if err := v.Username("1bad"); err == nil {
		t.Fatal("Username() error = nil, want validation error")
	}
	if err := v.SearchID("0123"); err == nil {
		t.Fatal("SearchID() error = nil, want leading-zero error")
	}
}

func TestEnvValue(t *testing.T) {
	v := New()
	if err := v.EnvValue("LOG_LEVEL", "info"); err != nil {
		t.Fatalf("EnvValue() error = %v", err)
	}
	if err := v.EnvValue("bad-key", "x"); err == nil {
		t.Fatal("EnvValue() error = nil, want key validation error")
	}
	if err := v.EnvValue("KEY", "a\x00b"); err == nil {
		t.Fatal("EnvValue() error = nil, want null-byte error")
	}
}
