package service

import (
	"context"
	"sync"
	"time"
)

// SearchRateLimiter enforces per-user search rate limits.
// Owners are unlimited. Regular users default to 5 requests / minute.
type SearchRateLimiter struct {
	limit  int
	window time.Duration
	owners map[int64]struct{}

	mu      sync.Mutex
	events  map[int64][]time.Time
}

// NewSearchRateLimiter constructs a SearchRateLimiter.
func NewSearchRateLimiter(cfg Config) *SearchRateLimiter {
	cfg = cfg.withDefaults()
	return &SearchRateLimiter{
		limit:  cfg.SearchRateLimit,
		window: cfg.SearchRateWindow,
		owners: ownerSet(cfg.OwnerIDs),
		events: make(map[int64][]time.Time),
	}
}

// Allow reports whether userID may proceed.
func (r *SearchRateLimiter) Allow(ctx context.Context, userID int64) error {
	_ = ctx
	if userID <= 0 {
		return ErrInvalidInput
	}
	if _, ok := r.owners[userID]; ok {
		return nil
	}
	if r.limit <= 0 {
		return nil
	}

	now := time.Now()
	cutoff := now.Add(-r.window)

	r.mu.Lock()
	defer r.mu.Unlock()

	events := r.events[userID]
	kept := events[:0]
	for _, ts := range events {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	if len(kept) >= r.limit {
		r.events[userID] = kept
		return ErrRateLimited
	}
	r.events[userID] = append(kept, now)
	return nil
}
