package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/v3rsi/tgbot-versionx/internal/config"
	"github.com/v3rsi/tgbot-versionx/internal/database/lmdb"
	"github.com/v3rsi/tgbot-versionx/internal/database/sqlite"
)

// healthReport is the result of integration health checks.
type healthReport struct {
	OK     bool
	Issues []string
}

func (r healthReport) Error() string {
	if r.OK {
		return ""
	}
	return strings.Join(r.Issues, "; ")
}

// runHealthChecks verifies configuration, bot token, SQLite, and LMDB readiness.
func runHealthChecks(
	ctx context.Context,
	cfg *config.Config,
	sqlDB *sqlite.DatabaseManager,
	idDB, phoneDB, usernameDB *lmdb.DB,
) healthReport {
	var issues []string

	if cfg == nil {
		issues = append(issues, "configuration is nil")
	} else {
		if strings.TrimSpace(cfg.BotToken) == "" {
			issues = append(issues, "bot token is empty")
		} else if !strings.Contains(cfg.BotToken, ":") {
			issues = append(issues, "bot token format is invalid")
		}
		if len(cfg.BotOwnerIDs) == 0 {
			issues = append(issues, "bot owner IDs are empty")
		}
	}

	if sqlDB == nil {
		issues = append(issues, "sqlite is not initialized")
	} else {
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := sqlDB.Ping(pingCtx); err != nil {
			issues = append(issues, fmt.Sprintf("sqlite ping failed: %v", err))
		}
	}

	for name, db := range map[string]*lmdb.DB{
		"id":       idDB,
		"phone":    phoneDB,
		"username": usernameDB,
	} {
		if db == nil {
			issues = append(issues, fmt.Sprintf("lmdb %s is not initialized", name))
			continue
		}
		statsCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err := db.Stats(statsCtx)
		cancel()
		if err != nil {
			issues = append(issues, fmt.Sprintf("lmdb %s stats failed: %v", name, err))
		}
	}

	return healthReport{OK: len(issues) == 0, Issues: issues}
}
