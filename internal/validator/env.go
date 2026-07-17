package validator

import (
	"strings"
	"unicode/utf8"
)

// EnvValue validates a raw environment value for a named key.
func (v *Standard) EnvValue(key, value string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return Error{Field: "env", Message: "key is required"}
	}
	if !envKeyPattern.MatchString(key) {
		return Error{Field: key, Message: "key must match [A-Z][A-Z0-9_]*"}
	}
	if strings.ContainsRune(value, 0) {
		return Error{Field: key, Message: "value contains null bytes"}
	}
	if !utf8.ValidString(value) {
		return Error{Field: key, Message: "value contains invalid UTF-8"}
	}
	if len(value) > 1<<20 {
		return Error{Field: key, Message: "value exceeds 1 MiB"}
	}
	return nil
}
