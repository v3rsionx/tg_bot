package service

import (
	"sync"
	"sync/atomic"
	"time"
)

// AppStatistics tracks business-level search and points metrics.
type AppStatistics struct {
	TotalSearches       uint64
	TodaySearches       uint64
	SuccessfulSearches  uint64
	FailedSearches      uint64
	AverageLatency      time.Duration
	PointsSpent         int64
	RegisteredUsers     int64
	BannedUsers         int64
}

// statsAccumulator stores process-local business metrics.
type statsAccumulator struct {
	mu             sync.Mutex
	day            string
	todaySearches  uint64
	totalSearches  atomic.Uint64
	successful     atomic.Uint64
	failed         atomic.Uint64
	pointsSpent    atomic.Int64
	totalLatencyNS atomic.Int64
	registered     atomic.Int64
	banned         atomic.Int64
}

// newStatsAccumulator constructs a metrics accumulator.
func newStatsAccumulator() *statsAccumulator {
	return &statsAccumulator{day: time.Now().UTC().Format("2006-01-02")}
}

// observeSearch records one finished search attempt.
func (s *statsAccumulator) observeSearch(found bool, latency time.Duration, points int64) {
	s.totalSearches.Add(1)
	s.totalLatencyNS.Add(latency.Nanoseconds())
	if found {
		s.successful.Add(1)
	} else {
		s.failed.Add(1)
	}
	if points > 0 {
		s.pointsSpent.Add(points)
	}

	day := time.Now().UTC().Format("2006-01-02")
	s.mu.Lock()
	if s.day != day {
		s.day = day
		s.todaySearches = 0
	}
	s.todaySearches++
	today := s.todaySearches
	s.mu.Unlock()
	_ = today
}

// snapshot returns a consistent metrics view.
func (s *statsAccumulator) snapshot() AppStatistics {
	total := s.totalSearches.Load()
	var avg time.Duration
	if total > 0 {
		avg = time.Duration(s.totalLatencyNS.Load()) / time.Duration(total)
	}

	day := time.Now().UTC().Format("2006-01-02")
	s.mu.Lock()
	if s.day != day {
		s.day = day
		s.todaySearches = 0
	}
	today := s.todaySearches
	s.mu.Unlock()

	return AppStatistics{
		TotalSearches:      total,
		TodaySearches:      today,
		SuccessfulSearches: s.successful.Load(),
		FailedSearches:     s.failed.Load(),
		AverageLatency:     avg,
		PointsSpent:        s.pointsSpent.Load(),
		RegisteredUsers:    s.registered.Load(),
		BannedUsers:        s.banned.Load(),
	}
}

// addRegistered increments the known registered-user counter.
func (s *statsAccumulator) addRegistered(delta int64) {
	s.registered.Add(delta)
}

// addBanned increments or decrements the banned-user counter.
func (s *statsAccumulator) addBanned(delta int64) {
	s.banned.Add(delta)
}
