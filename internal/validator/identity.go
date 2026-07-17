package validator

import (
	"strings"
	"unicode/utf8"
)

// Phone validates a phone number candidate.
func (v *Standard) Phone(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return Error{Field: "phone", Message: "is required"}
	}
	if !utf8.ValidString(value) {
		return Error{Field: "phone", Message: "contains invalid UTF-8"}
	}
	if len(value) > 32 {
		return Error{Field: "phone", Message: "exceeds maximum length"}
	}
	digits := 0
	for i, r := range value {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r == '+' && i == 0:
		case r == '-' || r == '(' || r == ')' || r == ' ' || r == '.':
		default:
			return Error{Field: "phone", Message: "contains invalid characters"}
		}
	}
	if digits < 7 || digits > 15 {
		return Error{Field: "phone", Message: "must contain 7 to 15 digits"}
	}
	return nil
}

// Username validates a Telegram-like username.
func (v *Standard) Username(value string) error {
	value = strings.TrimSpace(strings.TrimPrefix(value, "@"))
	value = strings.ToLower(value)
	if value == "" {
		return Error{Field: "username", Message: "is required"}
	}
	if !utf8.ValidString(value) {
		return Error{Field: "username", Message: "contains invalid UTF-8"}
	}
	// Telegram usernames are 5–32; allow 3–32 for imported legacy keys.
	if len(value) < 3 || len(value) > 32 {
		return Error{Field: "username", Message: "must be between 3 and 32 characters"}
	}
	for i, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
			if i == 0 {
				return Error{Field: "username", Message: "must start with a letter"}
			}
		case r == '_':
		default:
			return Error{Field: "username", Message: "may contain only letters, digits, and underscore"}
		}
	}
	return nil
}

// TelegramUserID validates a Telegram user ID.
func (v *Standard) TelegramUserID(id int64) error {
	if id <= 0 {
		return Error{Field: "telegram_user_id", Message: "must be a positive integer"}
	}
	return nil
}

// SearchID validates an exact search ID key.
func (v *Standard) SearchID(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return Error{Field: "search_id", Message: "is required"}
	}
	if !utf8.ValidString(value) {
		return Error{Field: "search_id", Message: "contains invalid UTF-8"}
	}
	if len(value) > 32 {
		return Error{Field: "search_id", Message: "must be at most 32 digits"}
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return Error{Field: "search_id", Message: "must contain only digits"}
		}
	}
	if value[0] == '0' && value != "0" {
		return Error{Field: "search_id", Message: "must not contain leading zeros"}
	}
	return nil
}

// Command validates a bot command name or slash-command.
func (v *Standard) Command(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return Error{Field: "command", Message: "is required"}
	}
	if !utf8.ValidString(value) {
		return Error{Field: "command", Message: "contains invalid UTF-8"}
	}
	value = strings.TrimPrefix(value, "/")
	if at := strings.IndexByte(value, '@'); at >= 0 {
		value = value[:at]
	}
	value = strings.ToLower(value)
	if value == "" || len(value) > 32 {
		return Error{Field: "command", Message: "must be 1 to 32 characters"}
	}
	for i, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
			if i == 0 {
				return Error{Field: "command", Message: "must start with a letter"}
			}
		case r == '_':
		default:
			return Error{Field: "command", Message: "may contain only letters, digits, and underscore"}
		}
	}
	return nil
}
