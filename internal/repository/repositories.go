package repository

import "context"

// Repositories groups all SQLite-backed repository ports.
type Repositories struct {
	Users         UserRepository
	Transactions  TransactionRepository
	SearchHistory SearchHistoryRepository
}

// TxFunc executes work inside a single database transaction.
// Implementations must commit on nil error and roll back otherwise.
type TxFunc func(ctx context.Context, repos Repositories) error

// Transactor starts and manages database transactions for repository work.
type Transactor interface {
	// WithinTx runs fn inside a transaction-scoped repository set.
	WithinTx(ctx context.Context, fn TxFunc) error
}
