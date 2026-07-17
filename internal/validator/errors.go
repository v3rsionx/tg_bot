package validator

import (
	"fmt"
	"strings"
)

// Error is a descriptive validation failure.
type Error struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e Error) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Errors aggregates multiple validation failures.
type Errors []Error

// Error implements the error interface.
func (e Errors) Error() string {
	if len(e) == 0 {
		return "validation failed"
	}
	parts := make([]string, 0, len(e))
	for _, item := range e {
		parts = append(parts, item.Error())
	}
	return strings.Join(parts, "; ")
}

// Add appends a field validation error.
func (e *Errors) Add(field, message string) {
	*e = append(*e, Error{Field: field, Message: message})
}

// Err returns nil when no validation errors were collected.
func (e Errors) Err() error {
	if len(e) == 0 {
		return nil
	}
	return e
}

// Unwrap returns the first error for errors.Is / errors.As compatibility chains.
func (e Errors) Unwrap() error {
	if len(e) == 0 {
		return nil
	}
	return e[0]
}
