package service

import (
	"fmt"
	"time"
)

const (
	defaultPointsPerSearch = 1
	defaultSearchRateLimit = 5
	defaultSearchRateWindow = time.Minute
	defaultHistoryLimit    = 10
)

// Config controls business-layer behavior.
type Config struct {
	// OwnerIDs are privileged Telegram user IDs.
	OwnerIDs []int64
	// PointsPerSearch is charged for each successful search.
	PointsPerSearch int
	// SearchRateLimit is the max user searches per SearchRateWindow.
	SearchRateLimit int
	// SearchRateWindow is the sliding window for user search rate limiting.
	SearchRateWindow time.Duration
	// DefaultHistoryLimit is used when callers pass a non-positive limit.
	DefaultHistoryLimit int
}

// Validate checks business configuration.
func (c Config) Validate() error {
	if c.PointsPerSearch < 0 {
		return fmt.Errorf("service: PointsPerSearch must be >= 0")
	}
	if c.SearchRateLimit < 0 {
		return fmt.Errorf("service: SearchRateLimit must be >= 0")
	}
	if c.SearchRateWindow < 0 {
		return fmt.Errorf("service: SearchRateWindow must be >= 0")
	}
	if c.DefaultHistoryLimit < 0 {
		return fmt.Errorf("service: DefaultHistoryLimit must be >= 0")
	}
	return nil
}

// withDefaults returns a copy of Config with production defaults applied.
func (c Config) withDefaults() Config {
	if c.PointsPerSearch == 0 {
		c.PointsPerSearch = defaultPointsPerSearch
	}
	if c.SearchRateLimit == 0 {
		c.SearchRateLimit = defaultSearchRateLimit
	}
	if c.SearchRateWindow == 0 {
		c.SearchRateWindow = defaultSearchRateWindow
	}
	if c.DefaultHistoryLimit == 0 {
		c.DefaultHistoryLimit = defaultHistoryLimit
	}
	return c
}

// ownerSet builds a lookup set from owner IDs.
func ownerSet(ownerIDs []int64) map[int64]struct{} {
	out := make(map[int64]struct{}, len(ownerIDs))
	for _, id := range ownerIDs {
		if id > 0 {
			out[id] = struct{}{}
		}
	}
	return out
}
