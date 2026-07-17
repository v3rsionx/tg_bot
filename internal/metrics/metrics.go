package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// Collector tracks operational counters and latencies.
type Collector struct {
	totalSearches      atomic.Int64
	successfulSearches atomic.Int64
	failedSearches     atomic.Int64
	todaySearches      atomic.Int64
	currentUsers       atomic.Int64
	cacheHits          atomic.Int64
	cacheMisses        atomic.Int64
	pointUsage         atomic.Int64

	searchLatencySumNs atomic.Int64
	searchLatencyCount atomic.Int64
	peakLatencyNs      atomic.Int64
	sqliteLatencySumNs atomic.Int64
	sqliteLatencyCount atomic.Int64
	lmdbLatencySumNs   atomic.Int64
	lmdbLatencyCount   atomic.Int64

	importRows         atomic.Int64
	importDurationNs   atomic.Int64

	mu       sync.Mutex
	todayKey string
	clock    func() time.Time
	runtime  RuntimeSampler
}

// Option configures a Collector.
type Option func(*Collector)

// WithClock overrides the clock (tests).
func WithClock(clock func() time.Time) Option {
	return func(c *Collector) { c.clock = clock }
}

// WithRuntimeSampler injects a runtime sampler.
func WithRuntimeSampler(s RuntimeSampler) Option {
	return func(c *Collector) { c.runtime = s }
}

// New constructs a thread-safe metrics collector.
func New(opts ...Option) *Collector {
	c := &Collector{
		clock:   func() time.Time { return time.Now().UTC() },
		runtime: DefaultRuntimeSampler{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	c.todayKey = c.clock().Format("2006-01-02")
	return c
}

func (c *Collector) rollDay() {
	day := c.clock().Format("2006-01-02")
	c.mu.Lock()
	if day != c.todayKey {
		c.todayKey = day
		c.todaySearches.Store(0)
	}
	c.mu.Unlock()
}

// ObserveSearch records a search attempt.
func (c *Collector) ObserveSearch(success bool, latency time.Duration) {
	c.rollDay()
	c.totalSearches.Add(1)
	c.todaySearches.Add(1)
	if success {
		c.successfulSearches.Add(1)
	} else {
		c.failedSearches.Add(1)
	}
	c.observeLatency(&c.searchLatencySumNs, &c.searchLatencyCount, &c.peakLatencyNs, latency)
}

// ObserveSQLite records a SQLite operation latency.
func (c *Collector) ObserveSQLite(latency time.Duration) {
	c.observeLatency(&c.sqliteLatencySumNs, &c.sqliteLatencyCount, nil, latency)
}

// ObserveLMDB records an LMDB operation latency.
func (c *Collector) ObserveLMDB(latency time.Duration) {
	c.observeLatency(&c.lmdbLatencySumNs, &c.lmdbLatencyCount, nil, latency)
}

// IncCacheHit increments cache hit count.
func (c *Collector) IncCacheHit() { c.cacheHits.Add(1) }

// IncCacheMiss increments cache miss count.
func (c *Collector) IncCacheMiss() { c.cacheMisses.Add(1) }

// AddPointUsage adds consumed points.
func (c *Collector) AddPointUsage(points int64) {
	if points > 0 {
		c.pointUsage.Add(points)
	}
}

// SetCurrentUsers sets the current known user count.
func (c *Collector) SetCurrentUsers(n int64) {
	c.currentUsers.Store(n)
}

// AddCurrentUsers adjusts the current user gauge.
func (c *Collector) AddCurrentUsers(delta int64) {
	c.currentUsers.Add(delta)
}

// ObserveImport records import throughput sample.
func (c *Collector) ObserveImport(rows int64, duration time.Duration) {
	if rows > 0 {
		c.importRows.Add(rows)
	}
	if duration > 0 {
		c.importDurationNs.Add(duration.Nanoseconds())
	}
}

func (c *Collector) observeLatency(sum, count, peak *atomic.Int64, latency time.Duration) {
	if latency < 0 {
		return
	}
	ns := latency.Nanoseconds()
	sum.Add(ns)
	count.Add(1)
	if peak == nil {
		return
	}
	for {
		cur := peak.Load()
		if ns <= cur {
			return
		}
		if peak.CompareAndSwap(cur, ns) {
			return
		}
	}
}

func avgDuration(sumNs, count int64) time.Duration {
	if count <= 0 {
		return 0
	}
	return time.Duration(sumNs / count)
}
