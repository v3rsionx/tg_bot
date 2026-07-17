package cache

import (
	"container/list"
	"sync"
	"time"

	"github.com/v3rsi/tgbot-versionx/internal/constants"
)

// Stats exposes cache counters.
type Stats struct {
	Hits      int64
	Misses    int64
	Size      int
	Evictions int64
	Expired   int64
}

// Options configures a Cache.
type Options struct {
	Capacity int
	TTL      time.Duration
	Clock    func() time.Time
}

// entry is one cache node.
type entry[V any] struct {
	key       string
	value     V
	expiresAt time.Time
}

// Cache is a generic TTL + LRU cache.
type Cache[V any] struct {
	mu        sync.Mutex
	capacity  int
	ttl       time.Duration
	clock     func() time.Time
	items     map[string]*list.Element
	lru       *list.List
	hits      int64
	misses    int64
	evictions int64
	expired   int64
}

// New constructs a Cache with TTL and LRU eviction.
func New[V any](opts Options) *Cache[V] {
	if opts.Capacity <= 0 {
		opts.Capacity = constants.DefaultCacheMaxEntries
	}
	if opts.TTL <= 0 {
		opts.TTL = constants.DefaultCacheTTL
	}
	if opts.Clock == nil {
		opts.Clock = func() time.Time { return time.Now().UTC() }
	}
	return &Cache[V]{
		capacity: opts.Capacity,
		ttl:      opts.TTL,
		clock:    opts.Clock,
		items:    make(map[string]*list.Element),
		lru:      list.New(),
	}
}

// Get returns a value when present and not expired.
func (c *Cache[V]) Get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var zero V
	el, ok := c.items[key]
	if !ok {
		c.misses++
		return zero, false
	}
	ent := el.Value.(*entry[V])
	if c.expiredLocked(ent) {
		c.removeElementLocked(el)
		c.expired++
		c.misses++
		return zero, false
	}
	c.lru.MoveToFront(el)
	c.hits++
	return ent.value, true
}

// Set stores a value with the configured TTL.
func (c *Cache[V]) Set(key string, value V) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL stores a value with an explicit TTL.
func (c *Cache[V]) SetWithTTL(key string, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ttl <= 0 {
		ttl = c.ttl
	}
	expires := c.clock().Add(ttl)
	if el, ok := c.items[key]; ok {
		ent := el.Value.(*entry[V])
		ent.value = value
		ent.expiresAt = expires
		c.lru.MoveToFront(el)
		return
	}
	el := c.lru.PushFront(&entry[V]{key: key, value: value, expiresAt: expires})
	c.items[key] = el
	c.evictLocked()
}

// Delete removes a key.
func (c *Cache[V]) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.removeElementLocked(el)
	}
}

// Invalidate is an alias for Delete.
func (c *Cache[V]) Invalidate(key string) { c.Delete(key) }

// InvalidateAll clears the entire cache.
func (c *Cache[V]) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*list.Element)
	c.lru.Init()
}

// Cleanup removes expired entries and returns how many were removed.
func (c *Cache[V]) Cleanup() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	removed := 0
	for el := c.lru.Back(); el != nil; {
		prev := el.Prev()
		ent := el.Value.(*entry[V])
		if c.expiredLocked(ent) {
			c.removeElementLocked(el)
			c.expired++
			removed++
		}
		el = prev
	}
	return removed
}

// Len returns the number of entries currently stored (including not-yet-cleaned expired).
func (c *Cache[V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lru.Len()
}

// Stats returns counters.
func (c *Cache[V]) Stats() Stats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return Stats{
		Hits:      c.hits,
		Misses:    c.misses,
		Size:      c.lru.Len(),
		Evictions: c.evictions,
		Expired:   c.expired,
	}
}

func (c *Cache[V]) expiredLocked(ent *entry[V]) bool {
	return !ent.expiresAt.IsZero() && !c.clock().Before(ent.expiresAt)
}

func (c *Cache[V]) evictLocked() {
	for c.lru.Len() > c.capacity {
		el := c.lru.Back()
		if el == nil {
			return
		}
		c.removeElementLocked(el)
		c.evictions++
	}
}

func (c *Cache[V]) removeElementLocked(el *list.Element) {
	ent := el.Value.(*entry[V])
	delete(c.items, ent.key)
	c.lru.Remove(el)
}
