package metrics

import (
	"encoding/json"
	"time"
)

// Snapshot is an exportable metrics view.
type Snapshot struct {
	TotalSearches      int64         `json:"total_searches"`
	SuccessfulSearches int64         `json:"successful_searches"`
	FailedSearches     int64         `json:"failed_searches"`
	TodaySearches      int64         `json:"today_searches"`
	CurrentUsers       int64         `json:"current_users"`
	AverageLatency     time.Duration `json:"average_latency"`
	PeakLatency        time.Duration `json:"peak_latency"`
	SQLiteLatency      time.Duration `json:"sqlite_latency"`
	LMDBLatency        time.Duration `json:"lmdb_latency"`
	CacheHits          int64         `json:"cache_hits"`
	CacheMisses        int64         `json:"cache_misses"`
	CacheHitRate       float64       `json:"cache_hit_rate"`
	PointUsage         int64         `json:"point_usage"`
	ImportSpeedRPS     float64       `json:"import_speed_rps"`
	MemoryAllocBytes   uint64        `json:"memory_alloc_bytes"`
	MemorySysBytes     uint64        `json:"memory_sys_bytes"`
	NumGoroutine       int           `json:"num_goroutine"`
	CPUCount           int           `json:"cpu_count"`
	CapturedAt         time.Time     `json:"captured_at"`
}

// Snapshot returns a point-in-time export of all tracked statistics.
func (c *Collector) Snapshot() Snapshot {
	c.rollDay()
	hits := c.cacheHits.Load()
	misses := c.cacheMisses.Load()
	var hitRate float64
	if total := hits + misses; total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	importRows := c.importRows.Load()
	importNs := c.importDurationNs.Load()
	var importRPS float64
	if importNs > 0 {
		importRPS = float64(importRows) / (float64(importNs) / float64(time.Second))
	}

	var memAlloc, memSys uint64
	var goroutines, cpus int
	if c.runtime != nil {
		sample := c.runtime.Sample()
		memAlloc = sample.AllocBytes
		memSys = sample.SysBytes
		goroutines = sample.Goroutines
		cpus = sample.CPUCount
	}

	return Snapshot{
		TotalSearches:      c.totalSearches.Load(),
		SuccessfulSearches: c.successfulSearches.Load(),
		FailedSearches:     c.failedSearches.Load(),
		TodaySearches:      c.todaySearches.Load(),
		CurrentUsers:       c.currentUsers.Load(),
		AverageLatency:     avgDuration(c.searchLatencySumNs.Load(), c.searchLatencyCount.Load()),
		PeakLatency:        time.Duration(c.peakLatencyNs.Load()),
		SQLiteLatency:      avgDuration(c.sqliteLatencySumNs.Load(), c.sqliteLatencyCount.Load()),
		LMDBLatency:        avgDuration(c.lmdbLatencySumNs.Load(), c.lmdbLatencyCount.Load()),
		CacheHits:          hits,
		CacheMisses:        misses,
		CacheHitRate:       hitRate,
		PointUsage:         c.pointUsage.Load(),
		ImportSpeedRPS:     importRPS,
		MemoryAllocBytes:   memAlloc,
		MemorySysBytes:     memSys,
		NumGoroutine:       goroutines,
		CPUCount:           cpus,
		CapturedAt:         c.clock(),
	}
}

// ExportJSON returns the snapshot encoded as JSON.
func (c *Collector) ExportJSON() ([]byte, error) {
	return json.Marshal(c.Snapshot())
}

// Reset clears counters (primarily for tests).
func (c *Collector) Reset() {
	c.totalSearches.Store(0)
	c.successfulSearches.Store(0)
	c.failedSearches.Store(0)
	c.todaySearches.Store(0)
	c.currentUsers.Store(0)
	c.cacheHits.Store(0)
	c.cacheMisses.Store(0)
	c.pointUsage.Store(0)
	c.searchLatencySumNs.Store(0)
	c.searchLatencyCount.Store(0)
	c.peakLatencyNs.Store(0)
	c.sqliteLatencySumNs.Store(0)
	c.sqliteLatencyCount.Store(0)
	c.lmdbLatencySumNs.Store(0)
	c.lmdbLatencyCount.Store(0)
	c.importRows.Store(0)
	c.importDurationNs.Store(0)
	c.mu.Lock()
	c.todayKey = c.clock().Format("2006-01-02")
	c.mu.Unlock()
}
