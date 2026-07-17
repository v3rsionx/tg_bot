package converter

import "fmt"

// Error is a descriptive converter failure.
type Error struct {
	Op  string
	Err error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return "converter: " + e.Op
	}
	return fmt.Sprintf("converter: %s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying cause.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Op: op, Err: err}
}
