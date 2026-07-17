package security

import (
	"unicode"
	"unicode/utf8"
)

// PreventLMDBKeyCorruption validates a string LMDB key used by ID/phone/username indexes.
func (s *Standard) PreventLMDBKeyCorruption(key []byte) error {
	if len(key) == 0 {
		return Error{Field: "lmdb_key", Message: "must not be empty"}
	}
	if len(key) > MaxLMDBKeyBytes {
		return Error{Field: "lmdb_key", Message: "exceeds maximum LMDB key size"}
	}
	for _, b := range key {
		if b == 0 {
			return Error{Field: "lmdb_key", Message: "must not contain null bytes"}
		}
	}
	if !utf8.Valid(key) {
		return Error{Field: "lmdb_key", Message: "contains invalid UTF-8"}
	}
	for _, r := range string(key) {
		if unicode.IsControl(r) {
			return Error{Field: "lmdb_key", Message: "must not contain control characters"}
		}
	}
	return nil
}
