package service

import (
	"context"
	"time"

	"github.com/v3rsionx/tg_bot/internal/telegram"
)

// TelegramSearch adapts SearchService to telegram.Search.
type TelegramSearch struct {
	Service *SearchService
}

// ExactLookup executes the business search flow and maps the transport DTO.
func (a TelegramSearch) ExactLookup(ctx context.Context, userID int64, query string) (telegram.SearchResult, error) {
	if a.Service == nil {
		return telegram.SearchResult{}, ErrNotSupported
	}
	outcome, err := a.Service.ExactLookup(ctx, userID, query)
	if err != nil {
		return telegram.SearchResult{}, err
	}
	return telegram.SearchResult{
		Found:    outcome.Found,
		ID:       outcome.ID,
		Name:     outcome.Name,
		Phone:    outcome.Phone,
		Username: outcome.Username,
		Extras:   outcome.Extras,
	}, nil
}

// TelegramPoints adapts PointsService to telegram.Points.
type TelegramPoints struct {
	Service *PointsService
}

// Balance returns the user point balance.
func (a TelegramPoints) Balance(ctx context.Context, userID int64) (int64, error) {
	if a.Service == nil {
		return 0, ErrNotSupported
	}
	return a.Service.GetBalance(ctx, userID)
}

// TelegramHistory adapts HistoryService to telegram.History.
type TelegramHistory struct {
	Service *HistoryService
}

// Recent returns recent history items for transport rendering.
func (a TelegramHistory) Recent(ctx context.Context, userID int64, limit int) ([]telegram.HistoryItem, error) {
	if a.Service == nil {
		return nil, ErrNotSupported
	}
	entries, err := a.Service.LastSearches(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]telegram.HistoryItem, 0, len(entries))
	for _, entry := range entries {
		out = append(out, telegram.HistoryItem{
			Query:     entry.Keyword,
			QueryType: entry.SearchType,
			CreatedAt: entry.Timestamp.UTC().Format(time.RFC3339),
		})
	}
	return out, nil
}

// TelegramAuthorizer adapts AccessService to telegram.Authorizer.
type TelegramAuthorizer struct {
	Service *AccessService
}

// IsOwner reports whether userID is an owner.
func (a TelegramAuthorizer) IsOwner(ctx context.Context, userID int64) bool {
	if a.Service == nil {
		return false
	}
	return a.Service.IsOwner(ctx, userID)
}

// Authorize validates ordinary bot access.
func (a TelegramAuthorizer) Authorize(ctx context.Context, userID int64) error {
	if a.Service == nil {
		return ErrUnauthorized
	}
	return a.Service.Authorize(ctx, userID)
}

// TelegramRateLimiter adapts SearchRateLimiter to telegram.RateLimiter.
type TelegramRateLimiter struct {
	Limiter *SearchRateLimiter
}

// Allow enforces the configured search rate limit.
func (a TelegramRateLimiter) Allow(ctx context.Context, userID int64) error {
	if a.Limiter == nil {
		return nil
	}
	return a.Limiter.Allow(ctx, userID)
}

// TelegramDependencies builds telegram.Dependencies from a business Module.
func TelegramDependencies(module *Module, logger telegram.Logger) telegram.Dependencies {
	if logger == nil {
		logger = telegram.NopLogger{}
	}
	return telegram.Dependencies{
		Logger:      logger,
		Authorizer:  TelegramAuthorizer{Service: module.Access},
		RateLimiter: TelegramRateLimiter{Limiter: module.RateLimit},
		Points:      TelegramPoints{Service: module.Points},
		Search:      TelegramSearch{Service: module.Search},
		History:     TelegramHistory{Service: module.History},
	}
}

var (
	_ telegram.Search      = TelegramSearch{}
	_ telegram.Points      = TelegramPoints{}
	_ telegram.History     = TelegramHistory{}
	_ telegram.Authorizer  = TelegramAuthorizer{}
	_ telegram.RateLimiter = TelegramRateLimiter{}
)
