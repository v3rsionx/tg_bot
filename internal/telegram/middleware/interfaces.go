package middleware

import "context"

// Logger is the injectable logging contract used by middleware.
type Logger interface {
	// Debugf logs a debug-level message.
	Debugf(format string, args ...any)
	// Infof logs an info-level message.
	Infof(format string, args ...any)
	// Warnf logs a warning-level message.
	Warnf(format string, args ...any)
	// Errorf logs an error-level message.
	Errorf(format string, args ...any)
}

// Authorizer decides whether a Telegram user may interact with the bot.
type Authorizer interface {
	// Authorize validates ordinary bot access for userID.
	Authorize(ctx context.Context, userID int64) error
}

// RateLimiter provides request throttling decisions.
type RateLimiter interface {
	// Allow reports whether userID may proceed with the current update.
	Allow(ctx context.Context, userID int64) error
}
