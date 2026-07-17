package security

import "fmt"

// Error is a descriptive security rejection.
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
