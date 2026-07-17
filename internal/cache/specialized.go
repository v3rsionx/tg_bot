package cache

import (
	"fmt"
	"time"

	"github.com/v3rsionx/tg_bot/internal/constants"
)

// SearchCache caches search results by query key.
type SearchCache struct {
	inner *Cache[any]
}

// NewSearchCache constructs a search result cache.
func NewSearchCache(ttl time.Duration, capacity int) *SearchCache {
	if capacity <= 0 {
		capacity = constants.DefaultSearchCacheSize
	}
	return &SearchCache{inner: New[any](Options{TTL: ttl, Capacity: capacity})}
}

func searchKey(searchType, query string) string {
	return searchType + ":" + query
}

// Get returns a cached search payload.
func (c *SearchCache) Get(searchType, query string) (any, bool) {
	return c.inner.Get(searchKey(searchType, query))
}

// Set stores a search payload.
func (c *SearchCache) Set(searchType, query string, value any) {
	c.inner.Set(searchKey(searchType, query), value)
}

// Invalidate removes one search entry.
func (c *SearchCache) Invalidate(searchType, query string) {
	c.inner.Invalidate(searchKey(searchType, query))
}

// InvalidateAll clears search cache.
func (c *SearchCache) InvalidateAll() { c.inner.InvalidateAll() }

// Cleanup removes expired entries.
func (c *SearchCache) Cleanup() int { return c.inner.Cleanup() }

// Stats returns cache statistics.
func (c *SearchCache) Stats() Stats { return c.inner.Stats() }

// UserCache caches user-related payloads by Telegram user ID.
type UserCache struct {
	inner *Cache[any]
}

// NewUserCache constructs a user cache.
func NewUserCache(ttl time.Duration, capacity int) *UserCache {
	if capacity <= 0 {
		capacity = constants.DefaultUserCacheSize
	}
	return &UserCache{inner: New[any](Options{TTL: ttl, Capacity: capacity})}
}

func userKey(userID int64) string { return fmt.Sprintf("user:%d", userID) }

// Get returns a cached user payload.
func (c *UserCache) Get(userID int64) (any, bool) { return c.inner.Get(userKey(userID)) }

// Set stores a user payload.
func (c *UserCache) Set(userID int64, value any) { c.inner.Set(userKey(userID), value) }

// Invalidate removes one user entry.
func (c *UserCache) Invalidate(userID int64) { c.inner.Invalidate(userKey(userID)) }

// InvalidateAll clears user cache.
func (c *UserCache) InvalidateAll() { c.inner.InvalidateAll() }

// Cleanup removes expired entries.
func (c *UserCache) Cleanup() int { return c.inner.Cleanup() }

// Stats returns cache statistics.
func (c *UserCache) Stats() Stats { return c.inner.Stats() }

// AdminCache caches admin panel / admin lookup payloads.
type AdminCache struct {
	inner *Cache[any]
}

// NewAdminCache constructs an admin cache.
func NewAdminCache(ttl time.Duration, capacity int) *AdminCache {
	if capacity <= 0 {
		capacity = constants.DefaultAdminCacheSize
	}
	return &AdminCache{inner: New[any](Options{TTL: ttl, Capacity: capacity})}
}

// Get returns a cached admin payload.
func (c *AdminCache) Get(key string) (any, bool) { return c.inner.Get(key) }

// Set stores an admin payload.
func (c *AdminCache) Set(key string, value any) { c.inner.Set(key, value) }

// Invalidate removes one admin entry.
func (c *AdminCache) Invalidate(key string) { c.inner.Invalidate(key) }

// InvalidateAll clears admin cache.
func (c *AdminCache) InvalidateAll() { c.inner.InvalidateAll() }

// Cleanup removes expired entries.
func (c *AdminCache) Cleanup() int { return c.inner.Cleanup() }

// Stats returns cache statistics.
func (c *AdminCache) Stats() Stats { return c.inner.Stats() }
