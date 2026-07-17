package security

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeMessageRejectsOversizedAndControl(t *testing.T) {
	s := New()
	if _, err := s.SanitizeMessage("hello world"); err != nil {
		t.Fatalf("SanitizeMessage() error = %v", err)
	}
	if _, err := s.SanitizeMessage(strings.Repeat("a", DefaultMaxMessageBytes+1)); err == nil {
		t.Fatal("SanitizeMessage() error = nil, want oversized error")
	}
	if _, err := s.SanitizeMessage("bad\x01input"); err == nil {
		t.Fatal("SanitizeMessage() error = nil, want control-char error")
	}
}

func TestRejectMalformedIDAndUTF8(t *testing.T) {
	s := New()
	if err := s.RejectMalformedID("12345"); err != nil {
		t.Fatalf("RejectMalformedID() error = %v", err)
	}
	if err := s.RejectMalformedID("12ab"); err == nil {
		t.Fatal("RejectMalformedID() error = nil, want malformed error")
	}
	if err := s.RejectMalformedID("0123"); err == nil {
		t.Fatal("RejectMalformedID() error = nil, want leading-zero error")
	}
	if err := s.RejectInvalidUTF8("field", string([]byte{0xff, 0xfe})); err == nil {
		t.Fatal("RejectInvalidUTF8() error = nil, want UTF-8 error")
	}
}

func TestPathTraversalAndInjections(t *testing.T) {
	s := New()
	if _, err := s.PreventPathTraversal("SQLITE_PATH", "./data/bot.db"); err != nil {
		t.Fatalf("PreventPathTraversal() error = %v", err)
	}
	if _, err := s.PreventPathTraversal("SQLITE_PATH", "../etc/passwd"); err == nil {
		t.Fatal("PreventPathTraversal() error = nil, want traversal error")
	}
	if _, err := s.PreventPathTraversal("SQLITE_PATH", `..\windows\system32`); err == nil {
		t.Fatal("PreventPathTraversal() error = nil, want windows traversal error")
	}
	if err := s.PreventConfigInjection("BOT_TOKEN", "ok-value"); err != nil {
		t.Fatalf("PreventConfigInjection() error = %v", err)
	}
	if err := s.PreventConfigInjection("BOT_TOKEN", "x$(whoami)"); err == nil {
		t.Fatal("PreventConfigInjection() error = nil, want injection error")
	}
	if err := s.PreventSQLInjection("query", "safe search text"); err != nil {
		t.Fatalf("PreventSQLInjection() error = %v", err)
	}
	if err := s.PreventSQLInjection("query", "1' OR 1=1"); err == nil {
		t.Fatal("PreventSQLInjection() error = nil, want SQL pattern error")
	}
	if err := s.PreventSQLInjection("query", "O'Brien"); err != nil {
		t.Fatalf("PreventSQLInjection() unexpected error for apostrophe name: %v", err)
	}
}

func TestAllowedRoots(t *testing.T) {
	root := t.TempDir()
	s := NewWithRoots(root)
	inside := filepath.Join(root, "bot.db")
	if _, err := s.PreventPathTraversal("SQLITE_PATH", inside); err != nil {
		t.Fatalf("PreventPathTraversal(inside) error = %v", err)
	}
	outside := filepath.Join(filepath.Dir(root), "outside.db")
	if _, err := s.PreventPathTraversal("SQLITE_PATH", outside); err == nil {
		t.Fatal("PreventPathTraversal(outside) error = nil, want root confinement error")
	}
}

func TestLMDBKeyAndNormalize(t *testing.T) {
	s := New()
	if err := s.PreventLMDBKeyCorruption([]byte("123456")); err != nil {
		t.Fatalf("PreventLMDBKeyCorruption() error = %v", err)
	}
	if err := s.PreventLMDBKeyCorruption([]byte{0}); err == nil {
		t.Fatal("PreventLMDBKeyCorruption() error = nil, want null-byte error")
	}
	phone, err := s.NormalizePhone("+1 (555) 123-4567")
	if err != nil {
		t.Fatalf("NormalizePhone() error = %v", err)
	}
	if phone != "15551234567" {
		t.Fatalf("NormalizePhone() = %q, want %q", phone, "15551234567")
	}
	user, err := s.NormalizeUsername("@Valid_User")
	if err != nil {
		t.Fatalf("NormalizeUsername() error = %v", err)
	}
	if user != "valid_user" {
		t.Fatalf("NormalizeUsername() = %q, want %q", user, "valid_user")
	}
}
