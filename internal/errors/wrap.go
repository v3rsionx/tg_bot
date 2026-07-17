package errors

import (
	stderrors "errors"
	"fmt"
)

// Wrap annotates err with code/operation/message. Returns nil when err is nil.
func Wrap(err error, code, message, operation string, opts ...Option) error {
	if err == nil {
		return nil
	}
	opts = append([]Option{WithCause(err)}, opts...)
	return New(code, message, operation, opts...)
}

// Wrapf is Wrap with a formatted message.
func Wrapf(err error, code, operation, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return Wrap(err, code, fmt.Sprintf(format, args...), operation)
}

// Unwrap is a thin alias to the standard library.
func Unwrap(err error) error {
	return stderrors.Unwrap(err)
}

// Is reports whether err matches target in the chain.
func Is(err, target error) bool {
	return stderrors.Is(err, target)
}

// As finds the first error in err's chain that matches target.
func As(err error, target any) bool {
	return stderrors.As(err, target)
}

// AsAppError extracts *AppError from an error chain.
func AsAppError(err error) (*AppError, bool) {
	var app *AppError
	if !stderrors.As(err, &app) {
		return nil, false
	}
	return app, true
}

// CodeOf returns the error code when err is or wraps an AppError.
func CodeOf(err error) string {
	if app, ok := AsAppError(err); ok {
		return app.Code
	}
	return ""
}
