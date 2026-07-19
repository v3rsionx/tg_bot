package search_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/v3rsionx/tg_bot/internal/search"
)

// seedStores loads exact-lookup fixtures matching the importer payload format.
func seedStores(t *testing.T) search.Stores {
	t.Helper()

	idStore := newMemoryEngine()
	phoneStore := newMemoryEngine()
	usernameStore := newMemoryEngine()

	ctx := context.Background()
	payload := []byte("+15551110001\x00alice_one\x00Alice One\x00{}")
	if err := idStore.Put(ctx, []byte("1001"), payload); err != nil {
		t.Fatalf("Put(id) error = %v", err)
	}
	if err := phoneStore.Put(ctx, []byte("+15551110001"), []byte("1001")); err != nil {
		t.Fatalf("Put(phone) error = %v", err)
	}
	if err := usernameStore.Put(ctx, []byte("alice_one"), []byte("1001")); err != nil {
		t.Fatalf("Put(username) error = %v", err)
	}

	return search.Stores{
		ID:       idStore,
		Phone:    phoneStore,
		Username: usernameStore,
	}
}

// TestSearchByIDExactLookup verifies direct ID resolution.
func TestSearchByIDExactLookup(t *testing.T) {
	svc, err := search.New(search.Config{}, seedStores(t))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result, err := svc.SearchByID(context.Background(), "1001")
	if err != nil {
		t.Fatalf("SearchByID() error = %v", err)
	}
	if !result.Found || result.Record.Username != "alice_one" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Latency < 0 {
		t.Fatal("Latency must not be negative")
	}
}

// TestSearchByIDOptionalPhoneUsernameCombinations covers ID-centric payloads.
func TestSearchByIDOptionalPhoneUsernameCombinations(t *testing.T) {
	idStore := newMemoryEngine()
	phoneStore := newMemoryEngine()
	usernameStore := newMemoryEngine()
	ctx := context.Background()

	fixtures := []struct {
		id       string
		payload  []byte
		phone    string
		username string
		name     string
		extras   string
	}{
		{id: "2001", payload: []byte{0, 0, 0}},
		{id: "2002", payload: []byte("+15552220002\x00\x00\x00"), phone: "+15552220002"},
		{id: "2003", payload: []byte("\x00only_user\x00\x00"), username: "only_user"},
		{
			id:       "2004",
			payload:  []byte("+15552220004\x00both_user\x00\x00"),
			phone:    "+15552220004",
			username: "both_user",
		},
		{
			id:      "2005",
			payload: []byte("\x00\x00Fabiana\x00{\"access_hash\":\"1\"}"),
			name:    "Fabiana",
			extras:  `{"access_hash":"1"}`,
		},
	}
	for _, f := range fixtures {
		if err := idStore.Put(ctx, []byte(f.id), f.payload); err != nil {
			t.Fatalf("Put(id %s): %v", f.id, err)
		}
		if f.phone != "" {
			if err := phoneStore.Put(ctx, []byte(f.phone), []byte(f.id)); err != nil {
				t.Fatalf("Put(phone %s): %v", f.phone, err)
			}
		}
		if f.username != "" {
			if err := usernameStore.Put(ctx, []byte(f.username), []byte(f.id)); err != nil {
				t.Fatalf("Put(username %s): %v", f.username, err)
			}
		}
	}

	svc, err := search.New(search.Config{}, search.Stores{
		ID: idStore, Phone: phoneStore, Username: usernameStore,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	cases := []struct {
		name     string
		id       string
		phone    string
		username string
		recName  string
		extras   string
	}{
		{name: "id only", id: "2001"},
		{name: "id + phone", id: "2002", phone: "+15552220002"},
		{name: "id + username", id: "2003", username: "only_user"},
		{name: "id + phone + username", id: "2004", phone: "+15552220004", username: "both_user"},
		{name: "id + name + extras", id: "2005", recName: "Fabiana", extras: `{"access_hash":"1"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.SearchByID(ctx, tc.id)
			if err != nil {
				t.Fatalf("SearchByID() error = %v", err)
			}
			if !result.Found || result.Record.ID != tc.id ||
				result.Record.Phone != tc.phone || result.Record.Username != tc.username ||
				result.Record.Name != tc.recName || result.Record.Extras != tc.extras {
				t.Fatalf("unexpected result: %+v", result)
			}
		})
	}
}

// TestSearchByPhoneResolvesThroughID verifies phone -> id -> record.
func TestSearchByPhoneResolvesThroughID(t *testing.T) {
	svc, err := search.New(search.Config{}, seedStores(t))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result, err := svc.SearchByPhone(context.Background(), "+1 (555) 111-0001")
	if err != nil {
		t.Fatalf("SearchByPhone() error = %v", err)
	}
	if result.Record.ID != "1001" || result.Record.Phone != "+15551110001" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

// TestSearchByUsernameResolvesThroughID verifies username -> id -> record.
func TestSearchByUsernameResolvesThroughID(t *testing.T) {
	svc, err := search.New(search.Config{}, seedStores(t))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result, err := svc.SearchByUsername(context.Background(), "@Alice_One")
	if err != nil {
		t.Fatalf("SearchByUsername() error = %v", err)
	}
	if result.Record.ID != "1001" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

// TestSearchCacheHitAndInvalidation covers cache behavior.
func TestSearchCacheHitAndInvalidation(t *testing.T) {
	svc, err := search.New(search.Config{CacheTTL: time.Minute}, seedStores(t))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	first, err := svc.SearchByID(context.Background(), "1001")
	if err != nil {
		t.Fatalf("SearchByID(first) error = %v", err)
	}
	if first.CacheHit {
		t.Fatal("first lookup should be a cache miss")
	}

	second, err := svc.SearchByID(context.Background(), "1001")
	if err != nil {
		t.Fatalf("SearchByID(second) error = %v", err)
	}
	if !second.CacheHit {
		t.Fatal("second lookup should be a cache hit")
	}

	svc.InvalidateID("1001")
	third, err := svc.SearchByID(context.Background(), "1001")
	if err != nil {
		t.Fatalf("SearchByID(third) error = %v", err)
	}
	if third.CacheHit {
		t.Fatal("lookup after invalidation should miss cache")
	}

	stats := svc.Stats()
	if stats.CacheHits < 1 || stats.Hits < 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

// TestSearchNotFoundAndInvalidQuery covers negative paths.
func TestSearchNotFoundAndInvalidQuery(t *testing.T) {
	svc, err := search.New(search.Config{}, seedStores(t))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = svc.SearchByID(context.Background(), "9999")
	if !errors.Is(err, search.ErrNotFound) {
		t.Fatalf("SearchByID(missing) error = %v, want ErrNotFound", err)
	}

	_, err = svc.SearchByPhone(context.Background(), "abc")
	if !errors.Is(err, search.ErrInvalidQuery) {
		t.Fatalf("SearchByPhone(invalid) error = %v, want ErrInvalidQuery", err)
	}

	stats := svc.Stats()
	if stats.Misses < 1 || stats.InvalidQueries < 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

// TestSearchRespectsContextTimeout verifies timeout handling.
func TestSearchRespectsContextTimeout(t *testing.T) {
	stores := seedStores(t)
	svc, err := search.New(search.Config{Timeout: time.Nanosecond}, stores)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond)

	_, err = svc.SearchByID(ctx, "1001")
	if err == nil {
		t.Fatal("SearchByID() error = nil, want timeout-related error")
	}
}
