package errors

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/v3rsi/tgbot-versionx/internal/constants"
)

// jsonError is the JSON serialization shape, including cause text.
type jsonError struct {
	Code      string       `json:"code"`
	Message   string       `json:"message"`
	Operation string       `json:"operation"`
	Cause     string       `json:"cause,omitempty"`
	Timestamp time.Time    `json:"timestamp"`
	Stack     []StackFrame `json:"stack,omitempty"`
}

// MarshalJSON implements json.Marshaler.
func (e *AppError) MarshalJSON() ([]byte, error) {
	if e == nil {
		return []byte("null"), nil
	}
	cause := e.causeText
	if cause == "" && e.Cause != nil {
		cause = e.Cause.Error()
	}
	return json.Marshal(jsonError{
		Code:      e.Code,
		Message:   e.Message,
		Operation: e.Operation,
		Cause:     cause,
		Timestamp: e.Timestamp,
		Stack:     e.Stack,
	})
}

// ToJSON returns a JSON encoding of the error.
func (e *AppError) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// LogFormat returns a single-line structured log representation.
func (e *AppError) LogFormat() string {
	if e == nil {
		return ""
	}
	cause := e.causeText
	if cause == "" && e.Cause != nil {
		cause = e.Cause.Error()
	}
	parts := []string{
		"code=" + e.Code,
		"op=" + e.Operation,
		"msg=" + quote(e.Message),
		"ts=" + e.Timestamp.Format(time.RFC3339Nano),
	}
	if cause != "" {
		parts = append(parts, "cause="+quote(cause))
	}
	if len(e.Stack) > 0 {
		parts = append(parts, "stack="+quote(strings.TrimSpace(formatStack(e.Stack))))
	}
	return strings.Join(parts, " ")
}

// TelegramSafeMessage returns a message safe to show in Telegram (no internals).
func (e *AppError) TelegramSafeMessage() string {
	if e == nil {
		return "An unexpected error occurred."
	}
	if e.userMessage != "" {
		return e.userMessage
	}
	switch e.Code {
	case constants.ErrCodeValidation:
		return "Invalid input. Please check your request and try again."
	case constants.ErrCodeSearch, constants.ErrCodeSearchNotFound:
		return "No results found."
	case constants.ErrCodeAuthorization, constants.ErrCodeForbidden:
		return "You are not allowed to perform this action."
	case constants.ErrCodeTimeout:
		return "The request timed out. Please try again."
	case constants.ErrCodeNetwork:
		return "A network error occurred. Please try again later."
	case constants.ErrCodeConfiguration:
		return "The service is temporarily unavailable."
	case constants.ErrCodeAdmin:
		return "Admin operation failed."
	default:
		return "Something went wrong. Please try again later."
	}
}

// UserFriendlyMessage returns a concise human-readable message.
func (e *AppError) UserFriendlyMessage() string {
	if e == nil {
		return "Unexpected error"
	}
	if e.userMessage != "" {
		return e.userMessage
	}
	if e.Message != "" {
		return e.Message
	}
	return e.TelegramSafeMessage()
}

// FormatError returns LogFormat for any error, preferring AppError.
func FormatError(err error) string {
	if err == nil {
		return ""
	}
	if app, ok := AsAppError(err); ok {
		return app.LogFormat()
	}
	return fmt.Sprintf("msg=%s", quote(err.Error()))
}

// TelegramSafe returns a Telegram-safe message for any error.
func TelegramSafe(err error) string {
	if err == nil {
		return ""
	}
	if app, ok := AsAppError(err); ok {
		return app.TelegramSafeMessage()
	}
	return "Something went wrong. Please try again later."
}

// UserFriendly returns a user-friendly message for any error.
func UserFriendly(err error) string {
	if err == nil {
		return ""
	}
	if app, ok := AsAppError(err); ok {
		return app.UserFriendlyMessage()
	}
	return "Unexpected error"
}

func quote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return `"` + s + `"`
}
