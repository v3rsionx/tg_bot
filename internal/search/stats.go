package search

import (
	"sync/atomic"
	"time"
)

// statsAccumulator tracks search metrics without global state.
type statsAccumulator struct {
	queries         atomic.Uint64
	hits            atomic.Uint64
	misses          atomic.Uint64
	cacheHits       atomic.Uint64
	cacheMisses     atomic.Uint64
	invalidQueries  atomic.Uint64
	errors          atomic.Uint64
	idQueries       atomic.Uint64
	phoneQueries    atomic.Uint64
	usernameQueries atomic.Uint64
	totalLatencyNS  atomic.Int64
}

// observe records one completed lookup.
func (s *statsAccumulator) observe(queryType QueryType, found, cacheHit bool, latency time.Duration, invalid, failed bool) {
	s.queries.Add(1)
	s.totalLatencyNS.Add(latency.Nanoseconds())

	switch queryType {
	case QueryTypeID:
		s.idQueries.Add(1)
	case QueryTypePhone:
		s.phoneQueries.Add(1)
	case QueryTypeUsername:
		s.usernameQueries.Add(1)
	}

	if invalid {
		s.invalidQueries.Add(1)
		return
	}
	if failed {
		s.errors.Add(1)
		return
	}
	if cacheHit {
		s.cacheHits.Add(1)
	} else {
		s.cacheMisses.Add(1)
	}
	if found {
		s.hits.Add(1)
	} else {
		s.misses.Add(1)
	}
}

// snapshot returns a consistent Statistics view.
func (s *statsAccumulator) snapshot() Statistics {
	queries := s.queries.Load()
	totalLatency := time.Duration(s.totalLatencyNS.Load())
	var average time.Duration
	if queries > 0 {
		average = totalLatency / time.Duration(queries)
	}
	return Statistics{
		Queries:         queries,
		Hits:            s.hits.Load(),
		Misses:          s.misses.Load(),
		CacheHits:       s.cacheHits.Load(),
		CacheMisses:     s.cacheMisses.Load(),
		InvalidQueries:  s.invalidQueries.Load(),
		Errors:          s.errors.Load(),
		IDQueries:       s.idQueries.Load(),
		PhoneQueries:    s.phoneQueries.Load(),
		UsernameQueries: s.usernameQueries.Load(),
		TotalLatency:    totalLatency,
		AverageLatency:  average,
	}
}
