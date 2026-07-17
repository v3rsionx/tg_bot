package errors

import (
	stderrors "errors"
	"encoding/json"
	"testing"

	"github.com/v3rsi/tgbot-versionx/internal/constants"
)

func TestWrapUnwrapIsAs(t *testing.T) {
	root := stderrors.New("disk full")
	err := SQLite("user.insert", "write failed", WithCause(root), WithStack())
	if !Is(err, root) {
		t.Fatal("Is(root) = false")
	}
	if Unwrap(err) != root {
		t.Fatal("Unwrap mismatch")
	}
	var app *AppError
	if !As(err, &app) {
		t.Fatal("As(*AppError) failed")
	}
	if app.Code != constants.ErrCodeSQLite {
		t.Fatalf("Code = %q", app.Code)
	}
	if len(app.Stack) == 0 {
		t.Fatal("expected stack frames")
	}
}

func TestJSONAndMessages(t *testing.T) {
	err := Validation("input.phone", "bad phone", WithUserMessage("Phone number is invalid"))
	data, jerr := err.ToJSON()
	if jerr != nil {
		t.Fatalf("ToJSON: %v", jerr)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if raw["code"] != constants.ErrCodeValidation {
		t.Fatalf("json code = %v", raw["code"])
	}
	if got := err.TelegramSafeMessage(); got != "Phone number is invalid" {
		t.Fatalf("TelegramSafeMessage = %q", got)
	}
	if got := err.UserFriendlyMessage(); got != "Phone number is invalid" {
		t.Fatalf("UserFriendlyMessage = %q", got)
	}
	if err.LogFormat() == "" {
		t.Fatal("LogFormat empty")
	}
}

func TestDomainConstructors(t *testing.T) {
	cases := []*AppError{
		Search("search.id", "failed"),
		SearchNotFound("search.id", "missing"),
		LMDB("lmdb.get", "failed"),
		Telegram("tg.send", "failed"),
		Admin("admin.ban", "failed"),
		Authorization("auth.check", "denied"),
		Forbidden("auth.check", "forbidden"),
		Configuration("config.load", "invalid"),
		Timeout("search.run", "deadline"),
		Network("tg.api", "unreachable"),
		Internal("runtime", "panic recovered"),
	}
	for _, c := range cases {
		if c.Code == "" || c.Operation == "" {
			t.Fatalf("incomplete error: %#v", c)
		}
		if TelegramSafe(c) == "" || UserFriendly(c) == "" {
			t.Fatalf("empty safe/friendly for %s", c.Code)
		}
	}
}

func TestWrapNil(t *testing.T) {
	if Wrap(nil, constants.ErrCodeInternal, "x", "y") != nil {
		t.Fatal("Wrap(nil) should return nil")
	}
}
