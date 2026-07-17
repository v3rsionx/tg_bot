package sqlite_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/v3rsionx/tg_bot/internal/database/sqlite"
	"github.com/v3rsionx/tg_bot/internal/models"
	"github.com/v3rsionx/tg_bot/internal/repository"
)

// TestDatabaseManagerRepositories covers migrations, CRUD, and transactions.
func TestDatabaseManagerRepositories(t *testing.T) {
	manager := openTestManager(t)
	defer func() {
		if err := manager.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	ctx := context.Background()
	repos := manager.Repositories()

	user := &models.User{
		ID:        1001,
		Username:  "alice",
		FirstName: "Alice",
		LastName:  "Example",
		Points:    10,
	}
	if err := repos.Users.Create(ctx, user); err != nil {
		t.Fatalf("Users.Create() error = %v", err)
	}

	gotUser, err := repos.Users.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Users.GetByID() error = %v", err)
	}
	if gotUser.Username != "alice" || gotUser.Points != 10 {
		t.Fatalf("unexpected user: %+v", gotUser)
	}

	txModel := &models.Transaction{
		UserID: user.ID,
		Amount: 5,
		Type:   models.TransactionTypeCredit,
		Reason: "bonus",
	}
	if err := repos.Transactions.Create(ctx, txModel); err != nil {
		t.Fatalf("Transactions.Create() error = %v", err)
	}
	if txModel.ID == 0 {
		t.Fatal("Transactions.Create() did not assign ID")
	}

	history := &models.SearchHistory{
		UserID:      user.ID,
		Query:       "5551234",
		QueryType:   "phone",
		ResultCount: 2,
		PointsSpent: 1,
	}
	if err := repos.SearchHistory.Create(ctx, history); err != nil {
		t.Fatalf("SearchHistory.Create() error = %v", err)
	}
	if history.ID == 0 {
		t.Fatal("SearchHistory.Create() did not assign ID")
	}

	err = manager.WithinTx(ctx, func(ctx context.Context, txRepos repository.Repositories) error {
		if err := txRepos.Users.UpdatePoints(ctx, user.ID, 9); err != nil {
			return err
		}
		return txRepos.Transactions.Create(ctx, &models.Transaction{
			UserID: user.ID,
			Amount: 1,
			Type:   models.TransactionTypeDebit,
			Reason: "search",
		})
	})
	if err != nil {
		t.Fatalf("WithinTx() error = %v", err)
	}

	gotUser, err = repos.Users.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Users.GetByID() after tx error = %v", err)
	}
	if gotUser.Points != 9 {
		t.Fatalf("points = %d, want 9", gotUser.Points)
	}

	err = manager.WithinTx(ctx, func(ctx context.Context, txRepos repository.Repositories) error {
		if err := txRepos.Users.UpdatePoints(ctx, user.ID, 8); err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	if err == nil {
		t.Fatal("WithinTx() error = nil, want rollback error")
	}

	gotUser, err = repos.Users.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Users.GetByID() after rollback error = %v", err)
	}
	if gotUser.Points != 9 {
		t.Fatalf("points after rollback = %d, want 9", gotUser.Points)
	}

	if _, err := repos.Users.GetByID(ctx, 999999); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("Users.GetByID() missing error = %v, want ErrNotFound", err)
	}
}

// openTestManager constructs an isolated DatabaseManager for tests.
func openTestManager(t *testing.T) *sqlite.DatabaseManager {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve module root: %v", err)
	}

	manager, err := sqlite.NewDatabaseManager(sqlite.Config{
		Path:           filepath.Join(t.TempDir(), "bot.db"),
		MigrationsPath: filepath.Join(root, "migrations"),
		MaxOpenConns:   1,
		MaxIdleConns:   1,
		BusyTimeout:    3 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewDatabaseManager() error = %v", err)
	}
	return manager
}
