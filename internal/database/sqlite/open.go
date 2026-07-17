package sqlite

import (
	"fmt"

	"github.com/v3rsionx/tg_bot/internal/config"
)

// OpenFromConfig constructs a DatabaseManager from validated application configuration.
// Callers own the returned manager and must Close it.
func OpenFromConfig(cfg *config.Config) (*DatabaseManager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("sqlite: config is nil")
	}

	return NewDatabaseManager(Config{
		Path:           cfg.SQLitePath,
		MigrationsPath: defaultMigrationsDir,
	})
}
