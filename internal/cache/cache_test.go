package cache

import (
	"testing"
	"time"
)

func TestTTLandLRU(t *testing.T) {
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	c := New[string](Options{
		Capacity: 2,
		TTL:      time.Minute,
		Clock:    func() time.Time { return now },
	})

	c.Set("a", "1")
	c.Set("b", "2")
	if _, ok := c.Get("a"); !ok {
		t.Fatal("expected hit for a")
	}
	c.Set("c", "3") // evicts LRU "b"
	if _, ok := c.Get("b"); ok {
		t.Fatal("expected b evicted")
	}
	if v, ok := c.Get("c"); !ok || v != "3" {
		t.Fatalf("c = %q ok=%v", v, ok)
	}

	now = now.Add(2 * time.Minute)
	if _, ok := c.Get("a"); ok {
		t.Fatal("expected a expired")
	}
	if n := c.Cleanup(); n < 0 {
		t.Fatalf("Cleanup = %d", n)
	}
	st := c.Stats()
	if st.Misses == 0 || st.Evictions == 0 {
		t.Fatalf("stats = %+v", st)
	}
}

func TestSpecializedCaches(t *testing.T) {
	sc := NewSearchCache(time.Minute, 10)
	sc.Set("id", "1", "payload")
	if v, ok := sc.Get("id", "1"); !ok || v != "payload" {
		t.Fatalf("SearchCache get = %v %v", v, ok)
	}
	sc.Invalidate("id", "1")
	if _, ok := sc.Get("id", "1"); ok {
		t.Fatal("expected invalidated")
	}

	uc := NewUserCache(time.Minute, 10)
	uc.Set(7, "user")
	if v, ok := uc.Get(7); !ok || v != "user" {
		t.Fatalf("UserCache get = %v %v", v, ok)
	}

	ac := NewAdminCache(time.Minute, 10)
	ac.Set("panel", "ok")
	if v, ok := ac.Get("panel"); !ok || v != "ok" {
		t.Fatalf("AdminCache get = %v %v", v, ok)
	}
	ac.InvalidateAll()
	if ac.Stats().Size != 0 {
		t.Fatal("expected empty admin cache")
	}
}
