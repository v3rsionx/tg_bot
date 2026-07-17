package repository

import (
	"context"

	"github.com/v3rsi/tgbot-versionx/internal/models"
)

// UserRepository defines persistence operations for users.
type UserRepository interface {
	// Create inserts a new user record.
	Create(ctx context.Context, user *models.User) error
	// Upsert inserts a user or updates mutable profile fields when the user already exists.
	Upsert(ctx context.Context, user *models.User) error
	// GetByID returns a user by Telegram user ID.
	GetByID(ctx context.Context, id int64) (*models.User, error)
	// Update replaces mutable user fields for an existing user.
	Update(ctx context.Context, user *models.User) error
	// UpdatePoints sets the absolute points balance for a user.
	UpdatePoints(ctx context.Context, userID int64, points int64) error
	// SetBanned updates the banned flag for a user.
	SetBanned(ctx context.Context, userID int64, banned bool) error
	// Exists reports whether a user with the given ID exists.
	Exists(ctx context.Context, id int64) (bool, error)
}
