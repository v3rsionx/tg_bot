package database

import (
	"context"
	"database/sql"
)

// Querier abstracts *sql.DB and *sql.Tx so repositories stay unit-test friendly.
type Querier interface {
	// ExecContext executes a statement without returning rows.
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	// QueryContext executes a query that returns rows.
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	// QueryRowContext executes a query that returns at most one row.
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// PrepareQuerier extends Querier with prepared-statement support.
type PrepareQuerier interface {
	Querier
	// PrepareContext creates a prepared statement for later execution.
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}
