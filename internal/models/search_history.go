package models

import "time"

// SearchHistory records a single search request performed by a user.
type SearchHistory struct {
	ID          int64
	UserID      int64
	Query       string
	QueryType   string
	ResultCount int
	PointsSpent int
	CreatedAt   time.Time
}
