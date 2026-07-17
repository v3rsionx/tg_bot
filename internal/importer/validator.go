package importer

import (
	"fmt"
	"strings"
	"unicode"
)

// Validator validates and normalizes parsed source fields.
type Validator struct {
	idColumn       int
	phoneColumn    int
	usernameColumn int
}

// NewValidator constructs a Validator using configured column indexes.
func NewValidator(cfg Config) *Validator {
	return &Validator{
		idColumn:       cfg.IDColumn,
		phoneColumn:    cfg.PhoneColumn,
		usernameColumn: cfg.UsernameColumn,
	}
}

// ValidateFields validates fields and returns a normalized Record.
func (v *Validator) ValidateFields(fields []string, meta Record) (Record, error) {
	maxCol := v.idColumn
	if v.phoneColumn > maxCol {
		maxCol = v.phoneColumn
	}
	if v.usernameColumn > maxCol {
		maxCol = v.usernameColumn
	}
	if len(fields) <= maxCol {
		return Record{}, fmt.Errorf("%w: insufficient columns", ErrInvalidRecord)
	}

	id := strings.TrimSpace(fields[v.idColumn])
	if !validID(id) {
		return Record{}, fmt.Errorf("%w: invalid id", ErrInvalidRecord)
	}

	phone := normalizePhone(fields[v.phoneColumn])
	if phone != "" && !validPhone(phone) {
		return Record{}, fmt.Errorf("%w: invalid phone", ErrInvalidRecord)
	}

	username := normalizeUsername(fields[v.usernameColumn])
	if username != "" && !validUsername(username) {
		return Record{}, fmt.Errorf("%w: invalid username", ErrInvalidRecord)
	}

	if phone == "" && username == "" {
		return Record{}, fmt.Errorf("%w: phone and username are both empty", ErrInvalidRecord)
	}

	meta.ID = id
	meta.Phone = phone
	meta.Username = username
	return meta, nil
}

// validID reports whether id is a positive numeric identifier.
func validID(id string) bool {
	if id == "" || len(id) > 32 {
		return false
	}
	for _, r := range id {
		if r < '0' || r > '9' {
			return false
		}
	}
	return id[0] != '0' || id == "0"
}

// normalizePhone keeps digits and an optional leading plus.
func normalizePhone(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(raw))
	for i, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
			continue
		}
		if r == '+' && i == 0 {
			b.WriteRune(r)
			continue
		}
		if unicode.IsSpace(r) || r == '-' || r == '(' || r == ')' {
			continue
		}
		// Keep unexpected characters so validation can reject the value.
		b.WriteRune(r)
	}
	return b.String()
}

// validPhone reports whether phone is a plausible E.164-like value.
func validPhone(phone string) bool {
	if phone == "" {
		return false
	}
	digits := phone
	if strings.HasPrefix(digits, "+") {
		digits = digits[1:]
	}
	if len(digits) < 7 || len(digits) > 15 {
		return false
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// normalizeUsername trims and lowercases a Telegram-like username.
func normalizeUsername(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "@")
	return strings.ToLower(raw)
}

// validUsername reports whether username matches Telegram-like constraints.
func validUsername(username string) bool {
	if len(username) < 3 || len(username) > 64 {
		return false
	}
	for i, r := range username {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '_':
		default:
			return false
		}
		if i == 0 && (r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}
