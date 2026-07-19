package handlers

import "context"

// Logger is the injectable logging contract used by handlers.
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

// Authorizer decides whether a Telegram user may perform privileged actions.
type Authorizer interface {
	// IsOwner reports whether userID is a bot owner.
	IsOwner(ctx context.Context, userID int64) bool
}

// Points exposes a read-only points balance port.
type Points interface {
	// Balance returns the current points balance for userID.
	Balance(ctx context.Context, userID int64) (int64, error)
}

// SearchResult is a transport DTO for exact-lookup responses.
type SearchResult struct {
	Found    bool
	ID       string
	Name     string
	Phone    string
	Username string
	Extras   string
}

// Search exposes an exact-lookup port used by text handlers.
type Search interface {
	// ExactLookup performs an exact lookup for query on behalf of userID.
	ExactLookup(ctx context.Context, userID int64, query string) (SearchResult, error)
}

// HistoryItem is a transport DTO for recent search history rows.
type HistoryItem struct {
	Query     string
	QueryType string
	CreatedAt string
}

// History exposes a read-only history port.
type History interface {
	// Recent returns the newest history items for userID.
	Recent(ctx context.Context, userID int64, limit int) ([]HistoryItem, error)
}

// Responder sends outbound Telegram messages without exposing bot internals.
type Responder interface {
	// ReplyText sends a plain-text reply to chatID.
	ReplyText(ctx context.Context, chatID int64, text string) error
	// AnswerCallback acknowledges a callback query.
	AnswerCallback(ctx context.Context, callbackID, text string) error
}
