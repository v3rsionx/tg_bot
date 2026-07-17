package validator

import (
	"strings"
	"unicode/utf8"
)

// TelegramToken validates a Telegram bot API token.
func (v *Standard) TelegramToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return Error{Field: "BOT_TOKEN", Message: "is required"}
	}
	if !utf8.ValidString(token) {
		return Error{Field: "BOT_TOKEN", Message: "contains invalid UTF-8"}
	}
	if strings.ContainsRune(token, 0) {
		return Error{Field: "BOT_TOKEN", Message: "contains null bytes"}
	}
	if !botTokenPattern.MatchString(token) {
		return Error{Field: "BOT_TOKEN", Message: "must match format <bot_id>:<secret>"}
	}
	return nil
}
