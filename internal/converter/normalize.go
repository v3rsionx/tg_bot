package converter

import (
	"strings"
	"unicode"
)

// normalizeHeader collapses a header for alias matching:
// case-insensitive, strips spaces, underscores, and dashes.
func normalizeHeader(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.TrimSpace(s) {
		if unicode.IsSpace(r) || r == '_' || r == '-' {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

var (
	idAliases = map[string]struct{}{
		"id": {}, "uid": {}, "telegramid": {}, "userid": {},
	}
	usernameAliases = map[string]struct{}{
		"username": {}, "user": {}, "login": {}, "nick": {}, "nickname": {}, "nik": {},
	}
	phoneAliases = map[string]struct{}{
		"phone": {}, "mobile": {}, "telephone": {}, "number": {}, "tel": {},
	}
	nameAliases = map[string]struct{}{
		"name": {}, "firstname": {}, "first": {},
	}
	lastNameAliases = map[string]struct{}{
		"lastname": {}, "surname": {}, "family": {}, "fname": {},
	}
)

func classifyHeader(raw string) (FieldRole, bool) {
	key := normalizeHeader(raw)
	switch {
	case keyIn(idAliases, key):
		return RoleID, true
	case keyIn(usernameAliases, key):
		return RoleUsername, true
	case keyIn(phoneAliases, key):
		return RolePhone, true
	case keyIn(lastNameAliases, key):
		return RoleLastName, true
	case keyIn(nameAliases, key):
		return RoleName, true
	default:
		return "", false
	}
}

func keyIn(m map[string]struct{}, key string) bool {
	_, ok := m[key]
	return ok
}
