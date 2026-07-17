package models

import "time"

// TransactionType classifies a points ledger movement.
type TransactionType string

const (
	// TransactionTypeCredit increases a user's point balance.
	TransactionTypeCredit TransactionType = "credit"
	// TransactionTypeDebit decreases a user's point balance.
	TransactionTypeDebit TransactionType = "debit"
)

// Transaction represents a durable points ledger entry.
type Transaction struct {
	ID        int64
	UserID    int64
	Amount    int64
	Type      TransactionType
	Reason    string
	CreatedAt time.Time
}
