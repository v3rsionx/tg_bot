package formatter

import "time"

// SearchResult is a single searchable record for display.
type SearchResult struct {
	ID       string
	Phone    string
	Username string
	Found    bool
	Type     string
	Latency  time.Duration
}

// Profile is a user profile view model.
type Profile struct {
	UserID   int64
	Username string
	Points   int64
	Banned   bool
	JoinedAt time.Time
}

// HistoryEntry is one history row.
type HistoryEntry struct {
	Query     string
	Type      string
	Found     bool
	CreatedAt time.Time
}

// Statistics is an aggregate stats view model.
type Statistics struct {
	TotalSearches      int64
	SuccessfulSearches int64
	FailedSearches     int64
	TodaySearches      int64
	CurrentUsers       int64
	AverageLatency     time.Duration
	PeakLatency        time.Duration
	CacheHitRate       float64
}

// AdminPanel is admin dashboard text content.
type AdminPanel struct {
	Title   string
	Lines   []string
	Footer  string
}
