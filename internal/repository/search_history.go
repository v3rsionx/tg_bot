package repository

import (
	"context"

	"github.com/v3rsionx/tg_bot/internal/models"
)

// SearchHistoryRepository defines persistence operations for search history.
type SearchHistoryRepository interface {
	// Create inserts a new search history record and assigns its generated ID.
	Create(ctx context.Context, entry *models.SearchHistory) error
	// GetByID returns a search history record by primary key.
	GetByID(ctx context.Context, id int64) (*models.SearchHistory, error)
	// ListByUserID returns search history for a user ordered by newest first.
	ListByUserID(ctx context.Context, userID int64, limit, offset int) ([]models.SearchHistory, error)
}
