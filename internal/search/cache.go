package search

import (
	"sync"
	"time"
)

// cacheEntry stores one cached lookup result.
type cacheEntry struct {
	record    Record
	found     bool
	expiresAt time.Time
}

// cache is a thread-safe TTL cache for exact-lookup results.
type cache struct {
	mu       sync.RWMutex
	ttl      time.Duration
	maxSize  int
	entries  map[string]cacheEntry
	enabled  bool
}

// newCache constructs a search cache from configuration.
func newCache(cfg Config) *cache {
	return &cache{
		ttl:     cfg.CacheTTL,
		maxSize: cfg.CacheMaxSize,
		entries: make(map[string]cacheEntry),
		enabled: cfg.cacheEnabled(),
	}
}

// cacheKey builds a namespaced cache key for an exact query.
func cacheKey(queryType QueryType, query string) string {
	return string(queryType) + ":" + query
}

// Get returns a cached record when present and not expired.
func (c *cache) Get(key string) (Record, bool, bool) {
	if c == nil || !c.enabled {
		return Record{}, false, false
	}

	now := time.Now()
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return Record{}, false, false
	}
	if !entry.expiresAt.IsZero() && now.After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return Record{}, false, false
	}
	return entry.record, entry.found, true
}

// Set stores a lookup outcome in the cache.
func (c *cache) Set(key string, record Record, found bool) {
	if c == nil || !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.maxSize > 0 && len(c.entries) >= c.maxSize {
		c.evictLocked()
	}

	var expiresAt time.Time
	if c.ttl > 0 {
		expiresAt = time.Now().Add(c.ttl)
	}
	c.entries[key] = cacheEntry{
		record:    record,
		found:     found,
		expiresAt: expiresAt,
	}
}

// Invalidate removes one cache key.
func (c *cache) Invalidate(key string) {
	if c == nil || !c.enabled {
		return
	}
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}

// InvalidateAll clears the entire cache.
func (c *cache) InvalidateAll() {
	if c == nil || !c.enabled {
		return
	}
	c.mu.Lock()
	c.entries = make(map[string]cacheEntry)
	c.mu.Unlock()
}

// InvalidateRecord removes ID/phone/username keys for one record.
func (c *cache) InvalidateRecord(record Record) {
	if c == nil || !c.enabled {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if record.ID != "" {
		delete(c.entries, cacheKey(QueryTypeID, record.ID))
	}
	if record.Phone != "" {
		delete(c.entries, cacheKey(QueryTypePhone, record.Phone))
	}
	if record.Username != "" {
		delete(c.entries, cacheKey(QueryTypeUsername, record.Username))
	}
}

// Len returns the number of cached entries.
func (c *cache) Len() int {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// evictLocked removes an arbitrary entry to keep the cache bounded.
func (c *cache) evictLocked() {
	now := time.Now()
	for key, entry := range c.entries {
		if !entry.expiresAt.IsZero() && now.After(entry.expiresAt) {
			delete(c.entries, key)
			if len(c.entries) < c.maxSize {
				return
			}
		}
	}
	for key := range c.entries {
		delete(c.entries, key)
		return
	}
}
