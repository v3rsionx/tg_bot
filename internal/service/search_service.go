package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/v3rsionx/tg_bot/internal/search"
)

// SearchOutcome is the structured business result of an exact lookup.
type SearchOutcome struct {
	Found     bool
	ID        string
	Phone     string
	Username  string
	Query     string
	QueryType string
	Latency   time.Duration
	PointsUsed int64
	CacheHit  bool
}

// SearchService orchestrates validation, points, exact lookup, and history.
type SearchService struct {
	cfg     Config
	engine  SearchEngine
	users   *UserService
	points  *PointsService
	history *HistoryService
	stats   *statsAccumulator
	log     Logger
}

// NewSearchService constructs a SearchService with dependency injection.
func NewSearchService(
	cfg Config,
	engine SearchEngine,
	users *UserService,
	points *PointsService,
	history *HistoryService,
	stats *statsAccumulator,
	log Logger,
) (*SearchService, error) {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if engine == nil {
		return nil, fmt.Errorf("service: SearchEngine is required")
	}
	if users == nil {
		return nil, fmt.Errorf("service: UserService is required")
	}
	if points == nil {
		return nil, fmt.Errorf("service: PointsService is required")
	}
	if history == nil {
		return nil, fmt.Errorf("service: HistoryService is required")
	}
	if stats == nil {
		stats = newStatsAccumulator()
	}
	if log == nil {
		log = NopLogger{}
	}
	return &SearchService{
		cfg:     cfg,
		engine:  engine,
		users:   users,
		points:  points,
		history: history,
		stats:   stats,
		log:     log,
	}, nil
}

// SearchByID performs the business search flow for an ID query.
func (s *SearchService) SearchByID(ctx context.Context, userID int64, id string) (SearchOutcome, error) {
	return s.execute(ctx, userID, strings.TrimSpace(id), search.QueryTypeID)
}

// SearchByPhone performs the business search flow for a phone query.
func (s *SearchService) SearchByPhone(ctx context.Context, userID int64, phone string) (SearchOutcome, error) {
	return s.execute(ctx, userID, strings.TrimSpace(phone), search.QueryTypePhone)
}

// SearchByUsername performs the business search flow for a username query.
func (s *SearchService) SearchByUsername(ctx context.Context, userID int64, username string) (SearchOutcome, error) {
	return s.execute(ctx, userID, strings.TrimSpace(username), search.QueryTypeUsername)
}

// ExactLookup auto-detects query type for transport adapters.
func (s *SearchService) ExactLookup(ctx context.Context, userID int64, query string) (SearchOutcome, error) {
	query = strings.TrimSpace(query)
	switch detectQueryType(query) {
	case search.QueryTypeID:
		return s.SearchByID(ctx, userID, query)
	case search.QueryTypePhone:
		return s.SearchByPhone(ctx, userID, query)
	default:
		return s.SearchByUsername(ctx, userID, query)
	}
}

// execute runs the shared search business flow.
func (s *SearchService) execute(ctx context.Context, userID int64, query string, queryType search.QueryType) (SearchOutcome, error) {
	start := time.Now()
	outcome := SearchOutcome{Query: query, QueryType: string(queryType)}

	if userID <= 0 {
		return outcome, fmt.Errorf("%w: user id", ErrInvalidInput)
	}
	if query == "" {
		return outcome, fmt.Errorf("%w: query", ErrInvalidInput)
	}

	if err := s.users.CreateUserIfNotExists(ctx, userID); err != nil {
		return outcome, err
	}
	user, err := s.users.Get(ctx, userID)
	if err != nil {
		return outcome, err
	}
	if user.IsBanned {
		return outcome, ErrBanned
	}

	cost := int64(s.cfg.PointsPerSearch)
	enough, err := s.points.HasEnoughPoints(ctx, userID, cost)
	if err != nil {
		return outcome, err
	}
	if !enough {
		return outcome, ErrInsufficientPoints
	}

	var result search.Result
	switch queryType {
	case search.QueryTypeID:
		result, err = s.engine.SearchByID(ctx, query)
	case search.QueryTypePhone:
		result, err = s.engine.SearchByPhone(ctx, query)
	case search.QueryTypeUsername:
		result, err = s.engine.SearchByUsername(ctx, query)
	default:
		return outcome, fmt.Errorf("%w: query type", ErrInvalidInput)
	}

	latency := time.Since(start)
	outcome.Latency = latency

	if err != nil {
		if errors.Is(err, search.ErrNotFound) {
			outcome.Found = false
			s.persistHistory(ctx, userID, query, string(queryType), false, latency, 0)
			s.stats.observeSearch(false, latency, 0)
			s.log.Infof("search miss user_id=%d type=%s query=%q latency=%s", userID, queryType, query, latency)
			return outcome, nil
		}
		if errors.Is(err, search.ErrInvalidQuery) {
			return outcome, fmt.Errorf("%w: %v", ErrInvalidInput, err)
		}
		s.log.Errorf("search error user_id=%d type=%s query=%q: %v", userID, queryType, query, err)
		return outcome, err
	}

	outcome.Found = result.Found
	outcome.CacheHit = result.CacheHit
	outcome.ID = result.Record.ID
	outcome.Phone = result.Record.Phone
	outcome.Username = result.Record.Username
	if outcome.Query == "" {
		outcome.Query = result.Query
	}

	pointsUsed := int64(0)
	if result.Found && cost > 0 {
		if err := s.points.RemovePoints(ctx, userID, cost, "search:"+string(queryType)); err != nil {
			return outcome, err
		}
		pointsUsed = cost
		outcome.PointsUsed = pointsUsed
	}

	s.persistHistory(ctx, userID, outcome.Query, string(queryType), result.Found, latency, int(pointsUsed))
	s.stats.observeSearch(result.Found, latency, pointsUsed)
	s.log.Infof(
		"search user_id=%d type=%s query=%q found=%t points=%d latency=%s",
		userID,
		queryType,
		outcome.Query,
		result.Found,
		pointsUsed,
		latency,
	)
	return outcome, nil
}

// persistHistory best-effort writes search history.
func (s *SearchService) persistHistory(
	ctx context.Context,
	userID int64,
	query, queryType string,
	found bool,
	latency time.Duration,
	points int,
) {
	if err := s.history.SaveSearch(ctx, HistoryEntry{
		UserID:      userID,
		SearchType:  queryType,
		Keyword:     query,
		ResultFound: found,
		Latency:     latency,
		PointsSpent: points,
		Timestamp:   time.Now().UTC(),
	}); err != nil {
		s.log.Errorf("save search history: %v", err)
	}
}

// detectQueryType chooses an exact-lookup index from raw user text.
func detectQueryType(query string) search.QueryType {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return search.QueryTypeUsername
	}
	if isAllDigits(trimmed) {
		return search.QueryTypeID
	}
	normalizedPhone := strings.Map(func(r rune) rune {
		switch {
		case r >= '0' && r <= '9':
			return r
		case r == '+' || r == '-' || r == '(' || r == ')' || unicode.IsSpace(r):
			return -1
		default:
			return r
		}
	}, trimmed)
	if strings.HasPrefix(trimmed, "+") || (isAllDigits(normalizedPhone) && len(normalizedPhone) >= 7) {
		return search.QueryTypePhone
	}
	return search.QueryTypeUsername
}

// isAllDigits reports whether value contains only decimal digits.
func isAllDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
