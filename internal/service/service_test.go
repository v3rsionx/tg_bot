package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/v3rsi/tgbot-versionx/internal/search"
	"github.com/v3rsi/tgbot-versionx/internal/service"
)

func newTestModule(t *testing.T, owners ...int64) (*service.Module, *memoryUsers, *memoryHistory, *stubSender) {
	t.Helper()
	users := newMemoryUsers()
	history := &memoryHistory{}
	sender := &stubSender{}
	engine := stubEngine{byID: map[string]search.Result{
		"1001": {
			Found: true,
			Record: search.Record{
				ID:       "1001",
				Phone:    "+15551110001",
				Username: "alice",
			},
			QueryType: search.QueryTypeID,
			Query:     "1001",
		},
	}}

	module, err := service.NewModule(service.Config{
		OwnerIDs:        owners,
		PointsPerSearch: 1,
		SearchRateLimit: 5,
	}, service.ModuleDeps{
		Users:        users,
		Transactions: &memoryTx{},
		History:      history,
		Engine:       engine,
		Sender:       sender,
		Logger:       service.NopLogger{},
	})
	if err != nil {
		t.Fatalf("NewModule() error = %v", err)
	}
	return module, users, history, sender
}

// TestSearchServiceConsumesPointsOnlyOnSuccess verifies the search business flow.
func TestSearchServiceConsumesPointsOnlyOnSuccess(t *testing.T) {
	module, _, history, _ := newTestModule(t, 1)
	ctx := context.Background()

	if err := module.Points.AddPoints(ctx, 42, 2, "seed"); err != nil {
		t.Fatalf("AddPoints() error = %v", err)
	}

	miss, err := module.Search.SearchByID(ctx, 42, "9999")
	if err != nil {
		t.Fatalf("SearchByID(miss) error = %v", err)
	}
	if miss.Found || miss.PointsUsed != 0 {
		t.Fatalf("unexpected miss result: %+v", miss)
	}
	balance, err := module.Points.GetBalance(ctx, 42)
	if err != nil || balance != 2 {
		t.Fatalf("balance after miss = %d err=%v, want 2", balance, err)
	}

	hit, err := module.Search.SearchByID(ctx, 42, "1001")
	if err != nil {
		t.Fatalf("SearchByID(hit) error = %v", err)
	}
	if !hit.Found || hit.PointsUsed != 1 || hit.Username != "alice" {
		t.Fatalf("unexpected hit result: %+v", hit)
	}
	balance, err = module.Points.GetBalance(ctx, 42)
	if err != nil || balance != 1 {
		t.Fatalf("balance after hit = %d err=%v, want 1", balance, err)
	}
	if len(history.data) != 2 {
		t.Fatalf("history rows = %d, want 2", len(history.data))
	}
}

// TestSearchServiceRejectsBannedAndPoorUsers verifies guard rails.
func TestSearchServiceRejectsBannedAndPoorUsers(t *testing.T) {
	module, _, _, _ := newTestModule(t, 1)
	ctx := context.Background()

	if _, err := module.Search.SearchByID(ctx, 7, "1001"); !errors.Is(err, service.ErrInsufficientPoints) {
		t.Fatalf("error = %v, want ErrInsufficientPoints", err)
	}

	if err := module.Points.AddPoints(ctx, 7, 5, "seed"); err != nil {
		t.Fatalf("AddPoints() error = %v", err)
	}
	if err := module.Admin.Ban(ctx, 1, 7); err != nil {
		t.Fatalf("Ban() error = %v", err)
	}
	if _, err := module.Search.SearchByID(ctx, 7, "1001"); !errors.Is(err, service.ErrBanned) {
		t.Fatalf("error = %v, want ErrBanned", err)
	}
}

// TestAdminServiceOwnerCommands verifies privileged operations and authorization.
func TestAdminServiceOwnerCommands(t *testing.T) {
	module, _, _, sender := newTestModule(t, 99)
	ctx := context.Background()

	if _, err := module.Admin.Execute(ctx, 1, "stats", nil); !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("non-owner error = %v, want ErrForbidden", err)
	}

	if err := module.Users.CreateUserIfNotExists(ctx, 10); err != nil {
		t.Fatalf("CreateUserIfNotExists() error = %v", err)
	}
	msg, err := module.Admin.Execute(ctx, 99, "addpoint", []string{"10", "3"})
	if err != nil {
		t.Fatalf("addpoint error = %v", err)
	}
	if msg == "" {
		t.Fatal("expected addpoint response")
	}
	balance, err := module.Points.GetBalance(ctx, 10)
	if err != nil || balance != 3 {
		t.Fatalf("balance = %d err=%v, want 3", balance, err)
	}

	if _, err := module.Admin.Execute(ctx, 99, "broadcast", []string{"hello"}); err != nil {
		t.Fatalf("broadcast error = %v", err)
	}
	if len(sender.messages) == 0 {
		t.Fatal("expected broadcast message")
	}

	statsMsg, err := module.Admin.Execute(ctx, 99, "stats", nil)
	if err != nil || statsMsg == "" {
		t.Fatalf("stats error=%v msg=%q", err, statsMsg)
	}
}

// TestHistoryClearAndCount verifies logical history clear semantics.
func TestHistoryClearAndCount(t *testing.T) {
	module, _, _, _ := newTestModule(t)
	ctx := context.Background()
	_ = module.History.SaveSearch(ctx, service.HistoryEntry{
		UserID: 5, SearchType: "id", Keyword: "1", ResultFound: true, Timestamp: time.Now().UTC(),
	})
	count, err := module.History.Count(ctx, 5)
	if err != nil || count != 1 {
		t.Fatalf("Count() = %d err=%v, want 1", count, err)
	}
	if err := module.History.Clear(ctx, 5); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}
	count, err = module.History.Count(ctx, 5)
	if err != nil || count != 0 {
		t.Fatalf("Count after clear = %d err=%v, want 0", count, err)
	}
}

// TestRateLimiterAllowsOwnersAndLimitsUsers verifies rate-limit rules.
func TestRateLimiterAllowsOwnersAndLimitsUsers(t *testing.T) {
	limiter := service.NewSearchRateLimiter(service.Config{
		OwnerIDs:         []int64{1},
		SearchRateLimit:  2,
		SearchRateWindow: time.Minute,
	})
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if err := limiter.Allow(ctx, 1); err != nil {
			t.Fatalf("owner Allow() error = %v", err)
		}
	}
	if err := limiter.Allow(ctx, 2); err != nil {
		t.Fatalf("first Allow() error = %v", err)
	}
	if err := limiter.Allow(ctx, 2); err != nil {
		t.Fatalf("second Allow() error = %v", err)
	}
	if err := limiter.Allow(ctx, 2); !errors.Is(err, service.ErrRateLimited) {
		t.Fatalf("third Allow() error = %v, want ErrRateLimited", err)
	}
}

// TestTelegramAdaptersMapDTOs verifies transport adapters.
func TestTelegramAdaptersMapDTOs(t *testing.T) {
	module, _, _, _ := newTestModule(t, 1)
	ctx := context.Background()
	_ = module.Points.AddPoints(ctx, 3, 1, "seed")

	searchPort := service.TelegramSearch{Service: module.Search}
	result, err := searchPort.ExactLookup(ctx, 3, "1001")
	if err != nil || !result.Found || result.ID != "1001" {
		t.Fatalf("ExactLookup() = %+v err=%v", result, err)
	}

	pointsPort := service.TelegramPoints{Service: module.Points}
	balance, err := pointsPort.Balance(ctx, 3)
	if err != nil || balance != 0 {
		t.Fatalf("Balance() = %d err=%v, want 0 after successful search", balance, err)
	}
}
