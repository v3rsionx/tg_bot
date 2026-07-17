package service

import (
	"context"

	"github.com/v3rsi/tgbot-versionx/internal/models"
	"github.com/v3rsi/tgbot-versionx/internal/repository"
	"github.com/v3rsi/tgbot-versionx/internal/search"
)

// UserRepository is the user persistence port used by business services.
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	Upsert(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id int64) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	UpdatePoints(ctx context.Context, userID int64, points int64) error
	SetBanned(ctx context.Context, userID int64, banned bool) error
	Exists(ctx context.Context, id int64) (bool, error)
}

// TransactionRepository is the points ledger persistence port.
type TransactionRepository interface {
	Create(ctx context.Context, tx *models.Transaction) error
	GetByID(ctx context.Context, id int64) (*models.Transaction, error)
	ListByUserID(ctx context.Context, userID int64, limit, offset int) ([]models.Transaction, error)
}

// HistoryRepository is the search-history persistence port.
type HistoryRepository interface {
	Create(ctx context.Context, entry *models.SearchHistory) error
	GetByID(ctx context.Context, id int64) (*models.SearchHistory, error)
	ListByUserID(ctx context.Context, userID int64, limit, offset int) ([]models.SearchHistory, error)
}

// Transactor runs multi-step writes atomically when available.
type Transactor interface {
	WithinTx(ctx context.Context, fn repository.TxFunc) error
}

// SearchEngine is the exact-lookup port; services must not access LMDB directly.
type SearchEngine interface {
	SearchByID(ctx context.Context, id string) (search.Result, error)
	SearchByPhone(ctx context.Context, phone string) (search.Result, error)
	SearchByUsername(ctx context.Context, username string) (search.Result, error)
	Stats() search.Statistics
}

// MessageSender delivers outbound user messages for broadcasts.
// Implementations belong to the composition root / transport adapters.
type MessageSender interface {
	// SendText delivers a plain-text message to chatID.
	SendText(ctx context.Context, chatID int64, text string) error
}

// UserDirectory lists known user IDs for admin broadcasts and counts.
type UserDirectory interface {
	// ListUserIDs returns known Telegram user IDs.
	ListUserIDs(ctx context.Context) ([]int64, error)
}
