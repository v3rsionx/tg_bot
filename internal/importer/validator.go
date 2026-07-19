package importer

import (
	"fmt"
	"strings"
	"sync"
	"unicode"
)

// Validator validates and normalizes parsed source fields.
type Validator struct {
	mu      sync.RWMutex
	mapping ColumnMapping
}

// NewValidator constructs a Validator using configured column indexes.
func NewValidator(cfg Config) *Validator {
	return &Validator{mapping: mappingFromConfig(cfg)}
}

// SetMapping replaces the active column mapping (per-file header resolution).
func (v *Validator) SetMapping(m ColumnMapping) {
	v.mu.Lock()
	v.mapping = m
	v.mu.Unlock()
}

// Mapping returns the active column mapping.
func (v *Validator) Mapping() ColumnMapping {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.mapping
}

// ValidateFields validates fields and returns a normalized Record.
func (v *Validator) ValidateFields(fields []string, meta Record) (Record, error) {
	m := v.Mapping()
	maxCol := m.ID
	for _, idx := range []int{m.Name, m.Phone, m.Username, m.Extras} {
		if idx > maxCol {
			maxCol = idx
		}
	}
	if m.ID < 0 {
		return Record{}, fmt.Errorf("%w: id column is not mapped", ErrInvalidRecord)
	}
	if len(fields) <= maxCol {
		return Record{}, fmt.Errorf("%w: insufficient columns", ErrInvalidRecord)
	}

	id := strings.TrimSpace(fields[m.ID])
	if !validID(id) {
		return Record{}, fmt.Errorf("%w: invalid id", ErrInvalidRecord)
	}

	// Invalid phone/username are dropped (cleared) so the ID row still imports.
	// Dirty dumps often contain usernames with spaces/symbols or short phones.
	phone := ""
	if m.Phone >= 0 {
		phone = normalizePhone(fields[m.Phone])
		if phone != "" && !validPhone(phone) {
			phone = ""
		}
	}

	username := ""
	if m.Username >= 0 {
		username = normalizeUsername(fields[m.Username])
		if username != "" && !validUsername(username) {
			// Strip emoji/spaces/symbols; keep a-z 0-9 _ when possible.
			username = sanitizeUsername(username)
			if username != "" && !validUsername(username) {
				username = ""
			}
		}
	}

	name := ""
	if m.Name >= 0 && m.Name < len(fields) {
		name = strings.TrimSpace(fields[m.Name])
	}

	extras := ""
	if m.Extras >= 0 && m.Extras < len(fields) {
		extras = strings.TrimSpace(fields[m.Extras])
		if extras == "" {
			extras = "{}"
		}
	}

	meta.ID = id
	meta.Name = name
	meta.Phone = phone
	meta.Username = username
	meta.Extras = extras
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

// sanitizeUsername keeps only Telegram-safe username characters.
func sanitizeUsername(username string) string {
	var b strings.Builder
	b.Grow(len(username))
	for _, r := range username {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_':
			b.WriteRune(r)
		}
	}
	return b.String()
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
