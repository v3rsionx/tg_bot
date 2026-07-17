package security

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// SanitizeText trims, validates UTF-8, and rejects oversized/control input.
func (s *Standard) SanitizeText(field, value string, maxBytes int) (string, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxMessageBytes
	}
	if err := s.RejectInvalidUTF8(field, value); err != nil {
		return "", err
	}
	if strings.ContainsRune(value, 0) {
		return "", Error{Field: field, Message: "contains null bytes"}
	}
	if len(value) > maxBytes {
		return "", Error{Field: field, Message: "exceeds maximum allowed size"}
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", Error{Field: field, Message: "is empty after sanitization"}
	}

	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		switch r {
		case '\n', '\t', '\r':
			b.WriteByte(' ')
			continue
		}
		if unicode.IsControl(r) {
			return "", Error{Field: field, Message: "contains control characters"}
		}
		if unicode.Is(unicode.Cf, r) {
			// Strip format characters (zero-width, bidi controls) used in spoofing.
			continue
		}
		b.WriteRune(r)
	}

	out := strings.Join(strings.Fields(b.String()), " ")
	if out == "" {
		return "", Error{Field: field, Message: "is empty after sanitization"}
	}
	if len(out) > maxBytes {
		return "", Error{Field: field, Message: "exceeds maximum allowed size"}
	}
	return out, nil
}

// SanitizeMessage sanitizes a user message with the configured size limit.
func (s *Standard) SanitizeMessage(value string) (string, error) {
	max := s.MaxMessageBytes
	if max <= 0 {
		max = DefaultMaxMessageBytes
	}
	return s.SanitizeText("message", value, max)
}

// RejectMalformedID rejects non-digit or oversized IDs.
func (s *Standard) RejectMalformedID(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return Error{Field: "id", Message: "is required"}
	}
	if !utf8.ValidString(value) {
		return Error{Field: "id", Message: "contains invalid UTF-8"}
	}
	if len(value) > 32 {
		return Error{Field: "id", Message: "is malformed or oversized"}
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return Error{Field: "id", Message: "is malformed"}
		}
	}
	if value[0] == '0' && value != "0" {
		return Error{Field: "id", Message: "is malformed"}
	}
	return nil
}

// RejectInvalidUTF8 rejects invalid UTF-8 sequences.
func (s *Standard) RejectInvalidUTF8(field, value string) error {
	if !utf8.ValidString(value) {
		return Error{Field: field, Message: "contains invalid UTF-8"}
	}
	return nil
}
