package repository

import (
	"context"

	"github.com/v3rsi/tgbot-versionx/internal/models"
)

// TransactionRepository defines persistence operations for point transactions.
type TransactionRepository interface {
	// Create inserts a new transaction and assigns its generated ID.
	Create(ctx context.Context, tx *models.Transaction) error
	// GetByID returns a transaction by primary key.
	GetByID(ctx context.Context, id int64) (*models.Transaction, error)
	// ListByUserID returns transactions for a user ordered by newest first.
	ListByUserID(ctx context.Context, userID int64, limit, offset int) ([]models.Transaction, error)
}
