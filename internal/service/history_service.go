package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/v3rsionx/tg_bot/internal/models"
)

// HistoryEntry is a business-level search history record.
type HistoryEntry struct {
	UserID      int64
	SearchType  string
	Keyword     string
	ResultFound bool
	Latency     time.Duration
	PointsSpent int
	Timestamp   time.Time
}

// HistoryService persists and reads search history.
type HistoryService struct {
	repo      HistoryRepository
	log       Logger
	clearedAt sync.Map
}

// NewHistoryService constructs a HistoryService with dependency injection.
func NewHistoryService(repo HistoryRepository, log Logger) *HistoryService {
	if log == nil {
		log = NopLogger{}
	}
	return &HistoryService{repo: repo, log: log}
}

// SaveSearch stores one search history row.
// ResultFound is persisted via ResultCount (1/0). Latency is logged and kept
// in the returned entry; durable latency columns are not available yet.
func (s *HistoryService) SaveSearch(ctx context.Context, entry HistoryEntry) error {
	if entry.UserID <= 0 || entry.Keyword == "" || entry.SearchType == "" {
		return fmt.Errorf("%w: history entry", ErrInvalidInput)
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	resultCount := 0
	if entry.ResultFound {
		resultCount = 1
	}
	model := &models.SearchHistory{
		UserID:      entry.UserID,
		Query:       entry.Keyword,
		QueryType:   entry.SearchType,
		ResultCount: resultCount,
		PointsSpent: entry.PointsSpent,
		CreatedAt:   entry.Timestamp,
	}
	if err := s.repo.Create(ctx, model); err != nil {
		return err
	}
	s.log.Infof(
		"search history user_id=%d type=%s keyword=%q found=%t latency=%s points=%d",
		entry.UserID,
		entry.SearchType,
		entry.Keyword,
		entry.ResultFound,
		entry.Latency,
		entry.PointsSpent,
	)
	return nil
}

// LastSearches returns the newest history rows for a user.
func (s *HistoryService) LastSearches(ctx context.Context, userID int64, limit int) ([]HistoryEntry, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("%w: user id", ErrInvalidInput)
	}
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.repo.ListByUserID(ctx, userID, limit, 0)
	if err != nil {
		return nil, err
	}
	cutoff, hasCutoff := s.clearCutoff(userID)
	out := make([]HistoryEntry, 0, len(rows))
	for _, row := range rows {
		if hasCutoff && !row.CreatedAt.After(cutoff) {
			continue
		}
		out = append(out, HistoryEntry{
			UserID:      row.UserID,
			SearchType:  row.QueryType,
			Keyword:     row.Query,
			ResultFound: row.ResultCount > 0,
			PointsSpent: row.PointsSpent,
			Timestamp:   row.CreatedAt,
		})
	}
	return out, nil
}

// Count returns the number of visible history rows for a user.
func (s *HistoryService) Count(ctx context.Context, userID int64) (int64, error) {
	if userID <= 0 {
		return 0, fmt.Errorf("%w: user id", ErrInvalidInput)
	}
	cutoff, hasCutoff := s.clearCutoff(userID)
	var total int64
	const page = 200
	for offset := 0; ; offset += page {
		rows, err := s.repo.ListByUserID(ctx, userID, page, offset)
		if err != nil {
			return 0, err
		}
		if len(rows) == 0 {
			break
		}
		for _, row := range rows {
			if hasCutoff && !row.CreatedAt.After(cutoff) {
				continue
			}
			total++
		}
		if len(rows) < page {
			break
		}
	}
	return total, nil
}

// Clear hides existing history for a user from subsequent reads.
func (s *HistoryService) Clear(ctx context.Context, userID int64) error {
	_ = ctx
	if userID <= 0 {
		return fmt.Errorf("%w: user id", ErrInvalidInput)
	}
	s.clearedAt.Store(userID, time.Now().UTC())
	s.log.Infof("search history cleared user_id=%d", userID)
	return nil
}

// Recent is a transport-friendly alias for LastSearches.
func (s *HistoryService) Recent(ctx context.Context, userID int64, limit int) ([]HistoryEntry, error) {
	return s.LastSearches(ctx, userID, limit)
}

// clearCutoff returns the logical clear timestamp for userID.
func (s *HistoryService) clearCutoff(userID int64) (time.Time, bool) {
	value, ok := s.clearedAt.Load(userID)
	if !ok {
		return time.Time{}, false
	}
	cutoff, ok := value.(time.Time)
	return cutoff, ok
}
