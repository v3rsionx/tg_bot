package sqlite

import (
	"context"
	"database/sql"
)

type txContextKey struct{}

// contextWithTx stores a transaction in ctx for repository methods to consume.
func contextWithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

// txFromContext returns the transaction attached to ctx, if any.
func txFromContext(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(txContextKey{}).(*sql.Tx)
	return tx, ok
}

// execStmt executes a prepared statement, binding it to the active transaction when present.
func execStmt(ctx context.Context, stmt *sql.Stmt, args ...any) (sql.Result, error) {
	if tx, ok := txFromContext(ctx); ok {
		return tx.StmtContext(ctx, stmt).ExecContext(ctx, args...)
	}
	return stmt.ExecContext(ctx, args...)
}

// queryStmt runs a prepared query, binding it to the active transaction when present.
func queryStmt(ctx context.Context, stmt *sql.Stmt, args ...any) (*sql.Rows, error) {
	if tx, ok := txFromContext(ctx); ok {
		return tx.StmtContext(ctx, stmt).QueryContext(ctx, args...)
	}
	return stmt.QueryContext(ctx, args...)
}

// queryRowStmt runs a prepared single-row query, binding it to the active transaction when present.
func queryRowStmt(ctx context.Context, stmt *sql.Stmt, args ...any) *sql.Row {
	if tx, ok := txFromContext(ctx); ok {
		return tx.StmtContext(ctx, stmt).QueryRowContext(ctx, args...)
	}
	return stmt.QueryRowContext(ctx, args...)
}
