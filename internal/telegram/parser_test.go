package telegram

import "testing"

// TestParseCommandNormalizesMentionsAndArgs verifies command parsing.
func TestParseCommandNormalizesMentionsAndArgs(t *testing.T) {
	parsed := ParseCommand("/start@MyBot arg1 arg2")
	if !parsed.IsCommand {
		t.Fatal("expected command")
	}
	if parsed.Name != "start" {
		t.Fatalf("Name = %q, want start", parsed.Name)
	}
	if len(parsed.Args) != 2 || parsed.Args[0] != "arg1" || parsed.Args[1] != "arg2" {
		t.Fatalf("Args = %#v", parsed.Args)
	}
}

// TestParseCommandRejectsPlainText verifies non-commands.
func TestParseCommandRejectsPlainText(t *testing.T) {
	parsed := ParseCommand("hello world")
	if parsed.IsCommand {
		t.Fatal("plain text should not be a command")
	}
}
