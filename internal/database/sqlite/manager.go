package sqlite

import (
	"context"
	"fmt"
	"sync"

	"github.com/v3rsionx/tg_bot/internal/repository"
)

// DatabaseManager coordinates connection lifecycle, migrations, and repositories.
type DatabaseManager struct {
	mu            sync.RWMutex
	conn          *ConnectionManager
	cfg           Config
	users         *UserRepository
	transactions  *TransactionRepository
	searchHistory *SearchHistoryRepository
	closed        bool
}

// NewDatabaseManager opens SQLite, runs migrations, and prepares repositories.
func NewDatabaseManager(cfg Config) (*DatabaseManager, error) {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	conn, err := NewConnectionManager(cfg)
	if err != nil {
		return nil, err
	}

	manager := &DatabaseManager{
		conn: conn,
		cfg:  cfg,
	}

	ctx := context.Background()
	if err := manager.Migrate(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}

	if err := manager.initRepositories(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return manager, nil
}

// Migrate applies all pending SQL migrations.
func (m *DatabaseManager) Migrate(ctx context.Context) error {
	migrator := NewMigrator(m.conn.DB(), m.cfg.MigrationsPath)
	if err := migrator.Up(ctx); err != nil {
		return fmt.Errorf("sqlite: automatic migration failed: %w", err)
	}
	return nil
}

// initRepositories constructs repository implementations with prepared statements.
func (m *DatabaseManager) initRepositories(ctx context.Context) error {
	db := m.conn.DB()

	users, err := NewUserRepository(ctx, db)
	if err != nil {
		return err
	}
	transactions, err := NewTransactionRepository(ctx, db)
	if err != nil {
		users.Close()
		return err
	}
	searchHistory, err := NewSearchHistoryRepository(ctx, db)
	if err != nil {
		users.Close()
		transactions.Close()
		return err
	}

	m.users = users
	m.transactions = transactions
	m.searchHistory = searchHistory
	return nil
}

// Users returns the user repository port.
func (m *DatabaseManager) Users() repository.UserRepository {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.users
}

// Transactions returns the transaction repository port.
func (m *DatabaseManager) Transactions() repository.TransactionRepository {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.transactions
}

// SearchHistory returns the search history repository port.
func (m *DatabaseManager) SearchHistory() repository.SearchHistoryRepository {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.searchHistory
}

// Repositories returns a grouped set of repository ports for dependency injection.
func (m *DatabaseManager) Repositories() repository.Repositories {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return repository.Repositories{
		Users:         m.users,
		Transactions:  m.transactions,
		SearchHistory: m.searchHistory,
	}
}

// Connection returns the underlying connection manager.
func (m *DatabaseManager) Connection() *ConnectionManager {
	return m.conn
}

// Ping verifies database connectivity.
func (m *DatabaseManager) Ping(ctx context.Context) error {
	return m.conn.Ping(ctx)
}

// Close releases prepared statements and closes the connection pool.
func (m *DatabaseManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}
	m.closed = true

	var firstErr error
	if m.users != nil {
		if err := m.users.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if m.transactions != nil {
		if err := m.transactions.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if m.searchHistory != nil {
		if err := m.searchHistory.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if err := m.conn.Close(); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// WithinTx runs fn inside a database transaction using the shared repository set.
func (m *DatabaseManager) WithinTx(ctx context.Context, fn repository.TxFunc) error {
	if fn == nil {
		return fmt.Errorf("sqlite: transaction function is nil")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return fmt.Errorf("sqlite: database is closed")
	}

	repos := repository.Repositories{
		Users:         m.users,
		Transactions:  m.transactions,
		SearchHistory: m.searchHistory,
	}

	db := m.conn.DB()
	if db == nil {
		return fmt.Errorf("sqlite: database is closed")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: begin transaction: %w", err)
	}

	txCtx := contextWithTx(ctx, tx)
	if err := fn(txCtx, repos); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("sqlite: rollback transaction: %w (original: %v)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqlite: commit transaction: %w", err)
	}
	return nil
}
