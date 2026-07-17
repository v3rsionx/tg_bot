package telegram

import "errors"

var (
	// ErrUnauthorized indicates the caller is not allowed to use the bot.
	ErrUnauthorized = errors.New("telegram: unauthorized")
	// ErrForbidden indicates the caller lacks privileges for an admin action.
	ErrForbidden = errors.New("telegram: forbidden")
	// ErrRateLimited indicates the caller exceeded the allowed request rate.
	ErrRateLimited = errors.New("telegram: rate limited")
	// ErrClosed indicates the bot has been shut down.
	ErrClosed = errors.New("telegram: closed")
	// ErrNilBot indicates the underlying Telegram client was not initialized.
	ErrNilBot = errors.New("telegram: bot client is nil")
)
