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

// TransactionRepository persists point transactions using prepared statements.
type TransactionRepository struct {
	db    *sql.DB
	stmts *transactionStatements
}

type transactionStatements struct {
	create      *sql.Stmt
	getByID     *sql.Stmt
	listByUser  *sql.Stmt
}

// NewTransactionRepository constructs a TransactionRepository and prepares its SQL statements.
func NewTransactionRepository(ctx context.Context, db *sql.DB) (*TransactionRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("sqlite: transaction repository database is nil")
	}

	stmts, err := prepareTransactionStatements(ctx, db)
	if err != nil {
		return nil, err
	}

	return &TransactionRepository{
		db:    db,
		stmts: stmts,
	}, nil
}

// prepareTransactionStatements prepares all SQL used by TransactionRepository.
func prepareTransactionStatements(ctx context.Context, db *sql.DB) (*transactionStatements, error) {
	stmts := &transactionStatements{}
	var err error

	stmts.create, err = db.PrepareContext(ctx, `
INSERT INTO transactions (user_id, amount, type, reason, created_at)
VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: prepare transaction create: %w", err)
	}

	stmts.getByID, err = db.PrepareContext(ctx, `
SELECT id, user_id, amount, type, reason, created_at
FROM transactions
WHERE id = ?`)
	if err != nil {
		closeStatements(stmts.create)
		return nil, fmt.Errorf("sqlite: prepare transaction getByID: %w", err)
	}

	stmts.listByUser, err = db.PrepareContext(ctx, `
SELECT id, user_id, amount, type, reason, created_at
FROM transactions
WHERE user_id = ?
ORDER BY id DESC
LIMIT ? OFFSET ?`)
	if err != nil {
		closeStatements(stmts.create, stmts.getByID)
		return nil, fmt.Errorf("sqlite: prepare transaction listByUser: %w", err)
	}

	return stmts, nil
}

// Create inserts a new transaction and assigns its generated ID.
func (r *TransactionRepository) Create(ctx context.Context, txModel *models.Transaction) error {
	if txModel == nil {
		return fmt.Errorf("sqlite: transaction is nil")
	}
	if txModel.CreatedAt.IsZero() {
		txModel.CreatedAt = time.Now().UTC()
	}

	result, err := execStmt(
		ctx,
		r.stmts.create,
		txModel.UserID,
		txModel.Amount,
		string(txModel.Type),
		txModel.Reason,
		formatTime(txModel.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("sqlite: create transaction: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("sqlite: transaction last insert id: %w", err)
	}
	txModel.ID = id
	return nil
}

// GetByID returns a transaction by primary key.
func (r *TransactionRepository) GetByID(ctx context.Context, id int64) (*models.Transaction, error) {
	row := queryRowStmt(ctx, r.stmts.getByID, id)

	var item models.Transaction
	var txType string
	var createdAt string
	err := row.Scan(
		&item.ID,
		&item.UserID,
		&item.Amount,
		&txType,
		&item.Reason,
		&createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: get transaction: %w", err)
	}

	item.Type = models.TransactionType(txType)
	item.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// ListByUserID returns transactions for a user ordered by newest first.
func (r *TransactionRepository) ListByUserID(ctx context.Context, userID int64, limit, offset int) ([]models.Transaction, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("sqlite: limit must be positive")
	}
	if offset < 0 {
		return nil, fmt.Errorf("sqlite: offset must be >= 0")
	}

	rows, err := queryStmt(ctx, r.stmts.listByUser, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list transactions: %w", err)
	}
	defer rows.Close()

	items := make([]models.Transaction, 0, limit)
	for rows.Next() {
		var item models.Transaction
		var txType string
		var createdAt string
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.Amount,
			&txType,
			&item.Reason,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("sqlite: scan transaction: %w", err)
		}
		item.Type = models.TransactionType(txType)
		item.CreatedAt, err = parseTime(createdAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate transactions: %w", err)
	}
	return items, nil
}

// Close releases prepared statements owned by the repository.
func (r *TransactionRepository) Close() error {
	if r == nil || r.stmts == nil {
		return nil
	}
	return closeStatements(r.stmts.create, r.stmts.getByID, r.stmts.listByUser)
}
