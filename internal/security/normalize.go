package security

import (
	"strings"
)

// NormalizePhone returns digits-only phone representation.
func (s *Standard) NormalizePhone(value string) (string, error) {
	if err := s.RejectInvalidUTF8("phone", value); err != nil {
		return "", err
	}
	if strings.ContainsRune(value, 0) {
		return "", Error{Field: "phone", Message: "contains null bytes"}
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", Error{Field: "phone", Message: "is required"}
	}

	var digits strings.Builder
	digits.Grow(len(value))
	for i, r := range value {
		switch {
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
		case r == '+' && i == 0:
		case r == '-' || r == '(' || r == ')' || r == ' ' || r == '.':
		default:
			return "", Error{Field: "phone", Message: "contains invalid characters"}
		}
	}
	out := digits.String()
	if len(out) < 7 || len(out) > 15 {
		return "", Error{Field: "phone", Message: "must contain 7 to 15 digits"}
	}
	return out, nil
}

// NormalizeUsername returns a lowercase username without leading @.
func (s *Standard) NormalizeUsername(value string) (string, error) {
	if err := s.RejectInvalidUTF8("username", value); err != nil {
		return "", err
	}
	if strings.ContainsRune(value, 0) {
		return "", Error{Field: "username", Message: "contains null bytes"}
	}
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "@")
	value = strings.ToLower(value)
	if value == "" {
		return "", Error{Field: "username", Message: "is required"}
	}
	if len(value) < 3 || len(value) > 32 {
		return "", Error{Field: "username", Message: "must be between 3 and 32 characters"}
	}
	for i, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
			if i == 0 {
				return "", Error{Field: "username", Message: "must start with a letter"}
			}
		case r == '_':
		default:
			return "", Error{Field: "username", Message: "contains invalid characters"}
		}
	}
	return value, nil
}
