package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/v3rsi/tgbot-versionx/internal/models"
	"github.com/v3rsi/tgbot-versionx/internal/repository"
)

// UserRepository persists users using prepared statements.
type UserRepository struct {
	db    *sql.DB
	stmts *userStatements
}

type userStatements struct {
	create       *sql.Stmt
	upsert       *sql.Stmt
	getByID      *sql.Stmt
	update       *sql.Stmt
	updatePoints *sql.Stmt
	setBanned    *sql.Stmt
	exists       *sql.Stmt
}

// NewUserRepository constructs a UserRepository and prepares its SQL statements.
func NewUserRepository(ctx context.Context, db *sql.DB) (*UserRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("sqlite: user repository database is nil")
	}

	stmts, err := prepareUserStatements(ctx, db)
	if err != nil {
		return nil, err
	}

	return &UserRepository{
		db:    db,
		stmts: stmts,
	}, nil
}

// prepareUserStatements prepares all SQL used by UserRepository.
func prepareUserStatements(ctx context.Context, db *sql.DB) (*userStatements, error) {
	stmts := &userStatements{}
	var err error

	stmts.create, err = db.PrepareContext(ctx, `
INSERT INTO users (id, username, first_name, last_name, points, is_banned, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: prepare user create: %w", err)
	}

	stmts.upsert, err = db.PrepareContext(ctx, `
INSERT INTO users (id, username, first_name, last_name, points, is_banned, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    username = excluded.username,
    first_name = excluded.first_name,
    last_name = excluded.last_name,
    updated_at = excluded.updated_at`)
	if err != nil {
		closeStatements(stmts.create)
		return nil, fmt.Errorf("sqlite: prepare user upsert: %w", err)
	}

	stmts.getByID, err = db.PrepareContext(ctx, `
SELECT id, username, first_name, last_name, points, is_banned, created_at, updated_at
FROM users
WHERE id = ?`)
	if err != nil {
		closeStatements(stmts.create, stmts.upsert)
		return nil, fmt.Errorf("sqlite: prepare user getByID: %w", err)
	}

	stmts.update, err = db.PrepareContext(ctx, `
UPDATE users
SET username = ?, first_name = ?, last_name = ?, points = ?, is_banned = ?, updated_at = ?
WHERE id = ?`)
	if err != nil {
		closeStatements(stmts.create, stmts.upsert, stmts.getByID)
		return nil, fmt.Errorf("sqlite: prepare user update: %w", err)
	}

	stmts.updatePoints, err = db.PrepareContext(ctx, `
UPDATE users
SET points = ?, updated_at = ?
WHERE id = ?`)
	if err != nil {
		closeStatements(stmts.create, stmts.upsert, stmts.getByID, stmts.update)
		return nil, fmt.Errorf("sqlite: prepare user updatePoints: %w", err)
	}

	stmts.setBanned, err = db.PrepareContext(ctx, `
UPDATE users
SET is_banned = ?, updated_at = ?
WHERE id = ?`)
	if err != nil {
		closeStatements(stmts.create, stmts.upsert, stmts.getByID, stmts.update, stmts.updatePoints)
		return nil, fmt.Errorf("sqlite: prepare user setBanned: %w", err)
	}

	stmts.exists, err = db.PrepareContext(ctx, `SELECT 1 FROM users WHERE id = ? LIMIT 1`)
	if err != nil {
		closeStatements(stmts.create, stmts.upsert, stmts.getByID, stmts.update, stmts.updatePoints, stmts.setBanned)
		return nil, fmt.Errorf("sqlite: prepare user exists: %w", err)
	}

	return stmts, nil
}

// Create inserts a new user record.
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	if user == nil {
		return fmt.Errorf("sqlite: user is nil")
	}
	now := time.Now().UTC()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}

	_, err := execStmt(
		ctx,
		r.stmts.create,
		user.ID,
		user.Username,
		user.FirstName,
		user.LastName,
		user.Points,
		boolToInt(user.IsBanned),
		formatTime(user.CreatedAt),
		formatTime(user.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("sqlite: create user: %w", err)
	}
	return nil
}

// Upsert inserts a user or updates mutable profile fields when the user already exists.
func (r *UserRepository) Upsert(ctx context.Context, user *models.User) error {
	if user == nil {
		return fmt.Errorf("sqlite: user is nil")
	}
	now := time.Now().UTC()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now

	_, err := execStmt(
		ctx,
		r.stmts.upsert,
		user.ID,
		user.Username,
		user.FirstName,
		user.LastName,
		user.Points,
		boolToInt(user.IsBanned),
		formatTime(user.CreatedAt),
		formatTime(user.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("sqlite: upsert user: %w", err)
	}
	return nil
}

// GetByID returns a user by Telegram user ID.
func (r *UserRepository) GetByID(ctx context.Context, id int64) (*models.User, error) {
	row := queryRowStmt(ctx, r.stmts.getByID, id)

	var user models.User
	var banned int
	var createdAt string
	var updatedAt string
	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.FirstName,
		&user.LastName,
		&user.Points,
		&banned,
		&createdAt,
		&updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: get user: %w", err)
	}

	user.IsBanned = banned == 1
	user.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return nil, err
	}
	user.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update replaces mutable user fields for an existing user.
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	if user == nil {
		return fmt.Errorf("sqlite: user is nil")
	}
	user.UpdatedAt = time.Now().UTC()

	result, err := execStmt(
		ctx,
		r.stmts.update,
		user.Username,
		user.FirstName,
		user.LastName,
		user.Points,
		boolToInt(user.IsBanned),
		formatTime(user.UpdatedAt),
		user.ID,
	)
	if err != nil {
		return fmt.Errorf("sqlite: update user: %w", err)
	}
	return ensureRowsAffected(result, "update user")
}

// UpdatePoints sets the absolute points balance for a user.
func (r *UserRepository) UpdatePoints(ctx context.Context, userID int64, points int64) error {
	result, err := execStmt(ctx, r.stmts.updatePoints, points, formatTime(time.Now().UTC()), userID)
	if err != nil {
		return fmt.Errorf("sqlite: update user points: %w", err)
	}
	return ensureRowsAffected(result, "update user points")
}

// SetBanned updates the banned flag for a user.
func (r *UserRepository) SetBanned(ctx context.Context, userID int64, banned bool) error {
	result, err := execStmt(ctx, r.stmts.setBanned, boolToInt(banned), formatTime(time.Now().UTC()), userID)
	if err != nil {
		return fmt.Errorf("sqlite: set user banned: %w", err)
	}
	return ensureRowsAffected(result, "set user banned")
}

// Exists reports whether a user with the given ID exists.
func (r *UserRepository) Exists(ctx context.Context, id int64) (bool, error) {
	var value int
	err := queryRowStmt(ctx, r.stmts.exists, id).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("sqlite: exists user: %w", err)
	}
	return true, nil
}

// Close releases prepared statements owned by the repository.
func (r *UserRepository) Close() error {
	if r == nil || r.stmts == nil {
		return nil
	}
	return closeStatements(
		r.stmts.create,
		r.stmts.upsert,
		r.stmts.getByID,
		r.stmts.update,
		r.stmts.updatePoints,
		r.stmts.setBanned,
		r.stmts.exists,
	)
}

// boolToInt converts a boolean into a SQLite INTEGER flag.
func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// ensureRowsAffected returns ErrNotFound when a write matched no rows.
func ensureRowsAffected(result sql.Result, operation string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("sqlite: %s rows affected: %w", operation, err)
	}
	if affected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// closeStatements closes prepared statements and returns the first error encountered.
func closeStatements(stmts ...*sql.Stmt) error {
	var firstErr error
	for _, stmt := range stmts {
		if stmt == nil {
			continue
		}
		if err := stmt.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
