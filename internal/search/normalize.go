package search

import (
	"fmt"
	"strings"
	"unicode"
)

// normalizeID trims and validates an exact ID query.
func normalizeID(raw string) (string, error) {
	id := strings.TrimSpace(raw)
	if id == "" || len(id) > 32 {
		return "", fmt.Errorf("%w: id", ErrInvalidQuery)
	}
	for _, r := range id {
		if r < '0' || r > '9' {
			return "", fmt.Errorf("%w: id", ErrInvalidQuery)
		}
	}
	if id[0] == '0' && id != "0" {
		return "", fmt.Errorf("%w: id", ErrInvalidQuery)
	}
	return id, nil
}

// normalizePhone trims formatting characters for exact phone lookup.
func normalizePhone(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("%w: phone", ErrInvalidQuery)
	}

	var b strings.Builder
	b.Grow(len(raw))
	for i, r := range raw {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '+' && i == 0:
			b.WriteRune(r)
		case unicode.IsSpace(r), r == '-', r == '(', r == ')':
			continue
		default:
			return "", fmt.Errorf("%w: phone", ErrInvalidQuery)
		}
	}

	phone := b.String()
	digits := phone
	if strings.HasPrefix(digits, "+") {
		digits = digits[1:]
	}
	if len(digits) < 7 || len(digits) > 15 {
		return "", fmt.Errorf("%w: phone", ErrInvalidQuery)
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return "", fmt.Errorf("%w: phone", ErrInvalidQuery)
		}
	}
	return phone, nil
}

// normalizeUsername trims and lowercases an exact username query.
func normalizeUsername(raw string) (string, error) {
	username := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(raw), "@"))
	if len(username) < 3 || len(username) > 64 {
		return "", fmt.Errorf("%w: username", ErrInvalidQuery)
	}
	for i, r := range username {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '_':
		default:
			return "", fmt.Errorf("%w: username", ErrInvalidQuery)
		}
		if i == 0 && r >= '0' && r <= '9' {
			return "", fmt.Errorf("%w: username", ErrInvalidQuery)
		}
	}
	return username, nil
}
