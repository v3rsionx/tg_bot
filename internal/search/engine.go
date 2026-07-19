package search

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/v3rsionx/tg_bot/internal/database/lmdb"
)

// Engine is the injectable exact-lookup search contract.
type Engine interface {
	// SearchByID performs an exact ID lookup.
	SearchByID(ctx context.Context, id string) (Result, error)
	// SearchByPhone performs phone -> id -> record exact lookup.
	SearchByPhone(ctx context.Context, phone string) (Result, error)
	// SearchByUsername performs username -> id -> record exact lookup.
	SearchByUsername(ctx context.Context, username string) (Result, error)
	// Stats returns cumulative search statistics.
	Stats() Statistics
	// InvalidateAll clears the entire search cache.
	InvalidateAll()
	// InvalidateID removes cached entries for one ID and its record fields.
	InvalidateID(id string)
	// InvalidatePhone removes cached entries for one phone number.
	InvalidatePhone(phone string)
	// InvalidateUsername removes cached entries for one username.
	InvalidateUsername(username string)
	// Close marks the engine closed for future lookups.
	Close() error
}

// Service is a thread-safe exact-lookup search engine.
type Service struct {
	cfg    Config
	stores Stores
	cache  *cache
	stats  *statsAccumulator

	mu     sync.RWMutex
	closed bool
}

// New constructs a search Service with dependency-injected stores.
func New(cfg Config, stores Stores) (*Service, error) {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if err := stores.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		cfg:    cfg,
		stores: stores,
		cache:  newCache(cfg),
		stats:  &statsAccumulator{},
	}, nil
}

// SearchByID performs an exact ID lookup.
func (s *Service) SearchByID(ctx context.Context, id string) (Result, error) {
	start := time.Now()
	normalized, err := normalizeID(id)
	if err != nil {
		s.stats.observe(QueryTypeID, false, false, time.Since(start), true, false)
		return Result{QueryType: QueryTypeID, Query: id, Latency: time.Since(start)}, err
	}
	return s.search(ctx, QueryTypeID, normalized, start, func(ctx context.Context) (Record, error) {
		return s.loadByID(ctx, normalized)
	})
}

// SearchByPhone performs phone -> id -> record exact lookup.
func (s *Service) SearchByPhone(ctx context.Context, phone string) (Result, error) {
	start := time.Now()
	normalized, err := normalizePhone(phone)
	if err != nil {
		s.stats.observe(QueryTypePhone, false, false, time.Since(start), true, false)
		return Result{QueryType: QueryTypePhone, Query: phone, Latency: time.Since(start)}, err
	}
	return s.search(ctx, QueryTypePhone, normalized, start, func(ctx context.Context) (Record, error) {
		idBytes, err := s.stores.Phone.Get(ctx, []byte(normalized))
		if err != nil {
			return Record{}, err
		}
		return s.loadByID(ctx, string(idBytes))
	})
}

// SearchByUsername performs username -> id -> record exact lookup.
func (s *Service) SearchByUsername(ctx context.Context, username string) (Result, error) {
	start := time.Now()
	normalized, err := normalizeUsername(username)
	if err != nil {
		s.stats.observe(QueryTypeUsername, false, false, time.Since(start), true, false)
		return Result{QueryType: QueryTypeUsername, Query: username, Latency: time.Since(start)}, err
	}
	return s.search(ctx, QueryTypeUsername, normalized, start, func(ctx context.Context) (Record, error) {
		idBytes, err := s.stores.Username.Get(ctx, []byte(normalized))
		if err != nil {
			return Record{}, err
		}
		return s.loadByID(ctx, string(idBytes))
	})
}

// search executes a cached exact lookup with timeout and statistics.
func (s *Service) search(
	ctx context.Context,
	queryType QueryType,
	query string,
	start time.Time,
	fetch func(context.Context) (Record, error),
) (Result, error) {
	if err := s.ensureOpen(); err != nil {
		s.stats.observe(queryType, false, false, time.Since(start), false, true)
		return Result{QueryType: queryType, Query: query, Latency: time.Since(start)}, err
	}

	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	key := cacheKey(queryType, query)
	if record, found, ok := s.cache.Get(key); ok {
		latency := time.Since(start)
		s.stats.observe(queryType, found, true, latency, false, false)
		result := Result{
			Record:    record,
			Found:     found,
			CacheHit:  true,
			QueryType: queryType,
			Query:     query,
			Latency:   latency,
		}
		if !found {
			return result, ErrNotFound
		}
		return result, nil
	}

	record, err := fetch(ctx)
	latency := time.Since(start)
	if err != nil {
		if isNotFound(err) {
			s.cache.Set(key, Record{}, false)
			s.stats.observe(queryType, false, false, latency, false, false)
			return Result{
				Found:     false,
				CacheHit:  false,
				QueryType: queryType,
				Query:     query,
				Latency:   latency,
			}, ErrNotFound
		}
		if isTimeout(err) {
			s.stats.observe(queryType, false, false, latency, false, true)
			return Result{QueryType: queryType, Query: query, Latency: latency}, fmt.Errorf("%w: %v", ErrTimeout, err)
		}
		s.stats.observe(queryType, false, false, latency, false, true)
		return Result{QueryType: queryType, Query: query, Latency: latency}, err
	}

	s.cache.Set(key, record, true)
	// Also cache by ID for subsequent direct lookups of the same record.
	if queryType != QueryTypeID && record.ID != "" {
		s.cache.Set(cacheKey(QueryTypeID, record.ID), record, true)
	}
	s.stats.observe(queryType, true, false, latency, false, false)
	return Result{
		Record:    record,
		Found:     true,
		CacheHit:  false,
		QueryType: queryType,
		Query:     query,
		Latency:   latency,
	}, nil
}

// loadByID loads and decodes a full record from the ID database.
func (s *Service) loadByID(ctx context.Context, id string) (Record, error) {
	payload, err := s.stores.ID.Get(ctx, []byte(id))
	if err != nil {
		return Record{}, err
	}
	return decodeIDPayload(id, payload)
}

// Stats returns cumulative search statistics.
func (s *Service) Stats() Statistics {
	return s.stats.snapshot()
}

// InvalidateAll clears the entire search cache.
func (s *Service) InvalidateAll() {
	s.cache.InvalidateAll()
}

// InvalidateID removes cached entries for one ID and its related fields.
func (s *Service) InvalidateID(id string) {
	normalized, err := normalizeID(id)
	if err != nil {
		s.cache.Invalidate(cacheKey(QueryTypeID, id))
		return
	}
	if record, found, ok := s.cache.Get(cacheKey(QueryTypeID, normalized)); ok && found {
		s.cache.InvalidateRecord(record)
		return
	}
	s.cache.Invalidate(cacheKey(QueryTypeID, normalized))
}

// InvalidatePhone removes cached entries for one phone number.
func (s *Service) InvalidatePhone(phone string) {
	normalized, err := normalizePhone(phone)
	if err != nil {
		s.cache.Invalidate(cacheKey(QueryTypePhone, phone))
		return
	}
	if record, found, ok := s.cache.Get(cacheKey(QueryTypePhone, normalized)); ok && found {
		s.cache.InvalidateRecord(record)
		return
	}
	s.cache.Invalidate(cacheKey(QueryTypePhone, normalized))
}

// InvalidateUsername removes cached entries for one username.
func (s *Service) InvalidateUsername(username string) {
	normalized, err := normalizeUsername(username)
	if err != nil {
		s.cache.Invalidate(cacheKey(QueryTypeUsername, username))
		return
	}
	if record, found, ok := s.cache.Get(cacheKey(QueryTypeUsername, normalized)); ok && found {
		s.cache.InvalidateRecord(record)
		return
	}
	s.cache.Invalidate(cacheKey(QueryTypeUsername, normalized))
}

// Close marks the engine closed for future lookups.
func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	s.cache.InvalidateAll()
	return nil
}

// ensureOpen verifies the service has not been closed.
func (s *Service) ensureOpen() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return ErrClosed
	}
	return nil
}

// withTimeout applies the configured timeout when the context has no deadline.
func (s *Service) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok || s.cfg.Timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, s.cfg.Timeout)
}

// isNotFound reports whether err represents a missing LMDB key.
func isNotFound(err error) bool {
	return errors.Is(err, lmdb.ErrNotFound) || errors.Is(err, ErrNotFound)
}

// isTimeout reports whether err represents a context deadline.
func isTimeout(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrTimeout)
}

var _ Engine = (*Service)(nil)
