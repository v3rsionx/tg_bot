package metrics

import (
	"encoding/json"
	"testing"
	"time"
)

type fixedRuntime struct{}

func (fixedRuntime) Sample() RuntimeSample {
	return RuntimeSample{AllocBytes: 100, SysBytes: 200, Goroutines: 3, CPUCount: 4}
}

func TestCollectorSnapshot(t *testing.T) {
	c := New(WithRuntimeSampler(fixedRuntime{}), WithClock(func() time.Time {
		return time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	}))
	c.ObserveSearch(true, 10*time.Millisecond)
	c.ObserveSearch(false, 30*time.Millisecond)
	c.ObserveSQLite(5 * time.Millisecond)
	c.ObserveLMDB(2 * time.Millisecond)
	c.IncCacheHit()
	c.IncCacheMiss()
	c.AddPointUsage(3)
	c.SetCurrentUsers(12)
	c.ObserveImport(1000, time.Second)

	snap := c.Snapshot()
	if snap.TotalSearches != 2 || snap.SuccessfulSearches != 1 || snap.FailedSearches != 1 {
		t.Fatalf("search counters = %+v", snap)
	}
	if snap.TodaySearches != 2 {
		t.Fatalf("TodaySearches = %d", snap.TodaySearches)
	}
	if snap.PeakLatency != 30*time.Millisecond {
		t.Fatalf("PeakLatency = %s", snap.PeakLatency)
	}
	if snap.CacheHitRate != 0.5 {
		t.Fatalf("CacheHitRate = %v", snap.CacheHitRate)
	}
	if snap.ImportSpeedRPS != 1000 {
		t.Fatalf("ImportSpeedRPS = %v", snap.ImportSpeedRPS)
	}
	if snap.MemoryAllocBytes != 100 || snap.CPUCount != 4 {
		t.Fatalf("runtime sample = %+v", snap)
	}

	data, err := c.ExportJSON()
	if err != nil {
		t.Fatalf("ExportJSON: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json: %v", err)
	}
}
