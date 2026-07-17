package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Migrator applies versioned SQL migration files from disk.
type Migrator struct {
	db   *sql.DB
	path string
}

// NewMigrator constructs a Migrator bound to a database and migrations directory.
func NewMigrator(db *sql.DB, migrationsPath string) *Migrator {
	return &Migrator{
		db:   db,
		path: migrationsPath,
	}
}

// Up applies all pending *.up.sql migrations in lexical version order.
func (m *Migrator) Up(ctx context.Context) error {
	if m.db == nil {
		return fmt.Errorf("sqlite: migrator database is nil")
	}
	if strings.TrimSpace(m.path) == "" {
		return fmt.Errorf("sqlite: migrations path is required")
	}

	if err := m.ensureMigrationsTable(ctx); err != nil {
		return err
	}

	files, err := m.listUpMigrations()
	if err != nil {
		return err
	}

	applied, err := m.appliedVersions(ctx)
	if err != nil {
		return err
	}

	for _, file := range files {
		version, err := parseMigrationVersion(file)
		if err != nil {
			return err
		}
		if _, exists := applied[version]; exists {
			continue
		}
		if err := m.applyUp(ctx, version, file); err != nil {
			return err
		}
	}

	return nil
}

// ensureMigrationsTable creates the schema_migrations bookkeeping table.
func (m *Migrator) ensureMigrationsTable(ctx context.Context) error {
	const query = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL
);`
	if _, err := m.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("sqlite: create schema_migrations: %w", err)
	}
	return nil
}

// listUpMigrations returns sorted absolute paths for *.up.sql files.
func (m *Migrator) listUpMigrations() ([]string, error) {
	entries, err := os.ReadDir(m.path)
	if err != nil {
		return nil, fmt.Errorf("sqlite: read migrations directory %q: %w", m.path, err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		files = append(files, filepath.Join(m.path, name))
	}
	sort.Strings(files)
	return files, nil
}

// appliedVersions returns the set of already applied migration versions.
func (m *Migrator) appliedVersions(ctx context.Context) (map[int]struct{}, error) {
	rows, err := m.db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list applied migrations: %w", err)
	}
	defer rows.Close()

	versions := make(map[int]struct{})
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("sqlite: scan migration version: %w", err)
		}
		versions[version] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate migration versions: %w", err)
	}
	return versions, nil
}

// applyUp executes one migration file inside a transaction and records its version.
func (m *Migrator) applyUp(ctx context.Context, version int, path string) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("sqlite: read migration %q: %w", path, err)
	}

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: begin migration %d: %w", version, err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, string(contents)); err != nil {
		return fmt.Errorf("sqlite: apply migration %d (%s): %w", version, filepath.Base(path), err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO schema_migrations (version, applied_at) VALUES (?, datetime('now'))`,
		version,
	); err != nil {
		return fmt.Errorf("sqlite: record migration %d: %w", version, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqlite: commit migration %d: %w", version, err)
	}
	return nil
}

// parseMigrationVersion extracts the leading integer version from a migration filename.
func parseMigrationVersion(path string) (int, error) {
	base := filepath.Base(path)
	parts := strings.SplitN(base, "_", 2)
	if len(parts) < 2 {
		return 0, fmt.Errorf("sqlite: invalid migration filename %q", base)
	}
	version, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("sqlite: invalid migration version in %q: %w", base, err)
	}
	return version, nil
}
