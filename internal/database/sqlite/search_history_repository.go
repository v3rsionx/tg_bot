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

// SearchHistoryRepository persists search history using prepared statements.
type SearchHistoryRepository struct {
	db    *sql.DB
	stmts *searchHistoryStatements
}

type searchHistoryStatements struct {
	create     *sql.Stmt
	getByID    *sql.Stmt
	listByUser *sql.Stmt
}

// NewSearchHistoryRepository constructs a SearchHistoryRepository and prepares its SQL statements.
func NewSearchHistoryRepository(ctx context.Context, db *sql.DB) (*SearchHistoryRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("sqlite: search history repository database is nil")
	}

	stmts, err := prepareSearchHistoryStatements(ctx, db)
	if err != nil {
		return nil, err
	}

	return &SearchHistoryRepository{
		db:    db,
		stmts: stmts,
	}, nil
}

// prepareSearchHistoryStatements prepares all SQL used by SearchHistoryRepository.
func prepareSearchHistoryStatements(ctx context.Context, db *sql.DB) (*searchHistoryStatements, error) {
	stmts := &searchHistoryStatements{}
	var err error

	stmts.create, err = db.PrepareContext(ctx, `
INSERT INTO search_history (user_id, query, query_type, result_count, points_spent, created_at)
VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: prepare search history create: %w", err)
	}

	stmts.getByID, err = db.PrepareContext(ctx, `
SELECT id, user_id, query, query_type, result_count, points_spent, created_at
FROM search_history
WHERE id = ?`)
	if err != nil {
		closeStatements(stmts.create)
		return nil, fmt.Errorf("sqlite: prepare search history getByID: %w", err)
	}

	stmts.listByUser, err = db.PrepareContext(ctx, `
SELECT id, user_id, query, query_type, result_count, points_spent, created_at
FROM search_history
WHERE user_id = ?
ORDER BY id DESC
LIMIT ? OFFSET ?`)
	if err != nil {
		closeStatements(stmts.create, stmts.getByID)
		return nil, fmt.Errorf("sqlite: prepare search history listByUser: %w", err)
	}

	return stmts, nil
}

// Create inserts a new search history record and assigns its generated ID.
func (r *SearchHistoryRepository) Create(ctx context.Context, entry *models.SearchHistory) error {
	if entry == nil {
		return fmt.Errorf("sqlite: search history is nil")
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}

	result, err := execStmt(
		ctx,
		r.stmts.create,
		entry.UserID,
		entry.Query,
		entry.QueryType,
		entry.ResultCount,
		entry.PointsSpent,
		formatTime(entry.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("sqlite: create search history: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("sqlite: search history last insert id: %w", err)
	}
	entry.ID = id
	return nil
}

// GetByID returns a search history record by primary key.
func (r *SearchHistoryRepository) GetByID(ctx context.Context, id int64) (*models.SearchHistory, error) {
	row := queryRowStmt(ctx, r.stmts.getByID, id)

	var entry models.SearchHistory
	var createdAt string
	err := row.Scan(
		&entry.ID,
		&entry.UserID,
		&entry.Query,
		&entry.QueryType,
		&entry.ResultCount,
		&entry.PointsSpent,
		&createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: get search history: %w", err)
	}

	entry.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// ListByUserID returns search history for a user ordered by newest first.
func (r *SearchHistoryRepository) ListByUserID(ctx context.Context, userID int64, limit, offset int) ([]models.SearchHistory, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("sqlite: limit must be positive")
	}
	if offset < 0 {
		return nil, fmt.Errorf("sqlite: offset must be >= 0")
	}

	rows, err := queryStmt(ctx, r.stmts.listByUser, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list search history: %w", err)
	}
	defer rows.Close()

	items := make([]models.SearchHistory, 0, limit)
	for rows.Next() {
		var entry models.SearchHistory
		var createdAt string
		if err := rows.Scan(
			&entry.ID,
			&entry.UserID,
			&entry.Query,
			&entry.QueryType,
			&entry.ResultCount,
			&entry.PointsSpent,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("sqlite: scan search history: %w", err)
		}
		entry.CreatedAt, err = parseTime(createdAt)
		if err != nil {
			return nil, err
		}
		items = append(items, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate search history: %w", err)
	}
	return items, nil
}

// Close releases prepared statements owned by the repository.
func (r *SearchHistoryRepository) Close() error {
	if r == nil || r.stmts == nil {
		return nil
	}
	return closeStatements(r.stmts.create, r.stmts.getByID, r.stmts.listByUser)
}
