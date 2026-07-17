package errors

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/v3rsi/tgbot-versionx/internal/constants"
)

// AppError is the production application error type.
type AppError struct {
	Code      string       `json:"code"`
	Message   string       `json:"message"`
	Operation string       `json:"operation"`
	Cause     error        `json:"-"`
	Timestamp time.Time    `json:"timestamp"`
	Stack     []StackFrame `json:"stack,omitempty"`

	userMessage string
	causeText   string
}

// Option configures AppError construction.
type Option func(*AppError)

// WithStack attaches a stack trace.
func WithStack() Option {
	return func(e *AppError) {
		e.Stack = CaptureStack(1)
	}
}

// WithUserMessage sets a user-facing message distinct from the internal message.
func WithUserMessage(msg string) Option {
	return func(e *AppError) {
		e.userMessage = msg
	}
}

// WithCause sets the underlying cause.
func WithCause(err error) Option {
	return func(e *AppError) {
		e.Cause = err
		if err != nil {
			e.causeText = err.Error()
		}
	}
}

// New constructs an AppError.
func New(code, message, operation string, opts ...Option) *AppError {
	e := &AppError{
		Code:      code,
		Message:   message,
		Operation: operation,
		Timestamp: time.Now().UTC(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
	return e
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %s: %v", e.Code, e.Operation, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s: %s", e.Code, e.Operation, e.Message)
}

// Unwrap returns the cause for errors.Unwrap / errors.Is chains.
func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// Is reports whether target matches this error by code when target is *AppError.
func (e *AppError) Is(target error) bool {
	if e == nil || target == nil {
		return false
	}
	t, ok := target.(*AppError)
	if !ok || t == nil {
		return false
	}
	return e.Code == t.Code
}

// Domain constructors.

func Validation(operation, message string, opts ...Option) *AppError {
	return New(constants.ErrCodeValidation, message, operation, opts...)
}

func Search(operation, message string, opts ...Option) *AppError {
	return New(constants.ErrCodeSearch, message, operation, opts...)
}

func SearchNotFound(operation, message string, opts ...Option) *AppError {
	return New(constants.ErrCodeSearchNotFound, message, operation, opts...)
}

func SQLite(operation, message string, opts ...Option) *AppError {
	return New(constants.ErrCodeSQLite, message, operation, opts...)
}

func LMDB(operation, message string, opts ...Option) *AppError {
	return New(constants.ErrCodeLMDB, message, operation, opts...)
}

func Telegram(operation, message string, opts ...Option) *AppError {
	return New(constants.ErrCodeTelegram, message, operation, opts...)
}

func Admin(operation, message string, opts ...Option) *AppError {
	return New(constants.ErrCodeAdmin, message, operation, opts...)
}

func Authorization(operation, message string, opts ...Option) *AppError {
	return New(constants.ErrCodeAuthorization, message, operation, opts...)
}

func Forbidden(operation, message string, opts ...Option) *AppError {
	return New(constants.ErrCodeForbidden, message, operation, opts...)
}

func Configuration(operation, message string, opts ...Option) *AppError {
	return New(constants.ErrCodeConfiguration, message, operation, opts...)
}

func Timeout(operation, message string, opts ...Option) *AppError {
	return New(constants.ErrCodeTimeout, message, operation, opts...)
}

func Network(operation, message string, opts ...Option) *AppError {
	return New(constants.ErrCodeNetwork, message, operation, opts...)
}

func Internal(operation, message string, opts ...Option) *AppError {
	return New(constants.ErrCodeInternal, message, operation, opts...)
}
