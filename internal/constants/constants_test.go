package constants

import "testing"

func TestCommandConstants(t *testing.T) {
	if CommandStart != "start" {
		t.Fatalf("CommandStart = %q", CommandStart)
	}
	if ErrCodeValidation == "" || ConfigBotToken == "" {
		t.Fatal("expected non-empty error/config constants")
	}
	if DefaultSearchTimeout <= 0 || DefaultCacheTTL <= 0 {
		t.Fatal("expected positive timeout defaults")
	}
}
