package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/v3rsi/tgbot-versionx/internal/security"
	"github.com/v3rsi/tgbot-versionx/internal/validator"
)

func validateConfig(cfg *Config, v validator.Validator, s security.Sanitizer) error {
	if cfg == nil {
		return validator.Error{Field: "config", Message: "is required"}
	}

	var errs validator.Errors

	if cfg.Version == "" {
		errs.Add("CONFIG_VERSION", "is required")
	} else if _, ok := SupportedVersions[cfg.Version]; !ok {
		errs.Add("CONFIG_VERSION", "unsupported configuration version")
	}

	if err := v.TelegramToken(cfg.BotToken); err != nil {
		errs.Add("BOT_TOKEN", fieldMessage(err))
	}
	if err := v.OwnerIDs(cfg.BotOwnerIDs); err != nil {
		errs.Add("BOT_OWNER_IDS", fieldMessage(err))
	}

	sqlitePath, err := s.PreventPathTraversal("SQLITE_PATH", cfg.SQLitePath)
	if err != nil {
		errs.Add("SQLITE_PATH", fieldMessage(err))
	} else if err := v.SQLitePath(sqlitePath); err != nil {
		errs.Add("SQLITE_PATH", fieldMessage(err))
	} else {
		cfg.SQLitePath = filepath.Clean(sqlitePath)
	}

	paths := map[string]*string{
		"LMDB_ID_PATH":       &cfg.LMDBIDPath,
		"LMDB_PHONE_PATH":    &cfg.LMDBPhonePath,
		"LMDB_USERNAME_PATH": &cfg.LMDBUsernamePath,
	}
	distinct := make(map[string]string, len(paths))
	for field, ptr := range paths {
		cleaned, err := s.PreventPathTraversal(field, *ptr)
		if err != nil {
			errs.Add(field, fieldMessage(err))
			continue
		}
		if err := v.LMDBPath(field, cleaned); err != nil {
			errs.Add(field, fieldMessage(err))
			continue
		}
		*ptr = filepath.Clean(cleaned)
		distinct[field] = *ptr
	}
	if len(distinct) == len(paths) {
		if err := v.DistinctPaths(distinct); err != nil {
			errs.Add("LMDB_PATHS", fieldMessage(err))
		}
	}

	if err := v.LogLevel(cfg.LogLevel); err != nil {
		errs.Add("LOG_LEVEL", fieldMessage(err))
	} else {
		cfg.LogLevel = normalizeLogLevel(cfg.LogLevel)
	}
	if err := v.WorkerCount(cfg.WorkerCount); err != nil {
		errs.Add("WORKER_COUNT", fieldMessage(err))
	}
	if err := v.BatchSize(cfg.BatchSize); err != nil {
		errs.Add("BATCH_SIZE", fieldMessage(err))
	}
	if err := v.Timeout("SEARCH_TIMEOUT", cfg.SearchTimeout); err != nil {
		errs.Add("SEARCH_TIMEOUT", fieldMessage(err))
	}
	if err := v.PositiveInt("POINTS_PER_SEARCH", cfg.PointsPerSearch); err != nil {
		errs.Add("POINTS_PER_SEARCH", fieldMessage(err))
	}
	if err := v.MaxResults(cfg.MaxResults); err != nil {
		errs.Add("MAX_RESULTS", fieldMessage(err))
	}
	if err := v.MaxResults(cfg.MaxSearchResult); err != nil {
		errs.Add("MAX_SEARCH_RESULT", fieldMessage(err))
	}
	if cfg.MaxResults != cfg.MaxSearchResult {
		errs.Add("MAX_RESULTS", "must match MAX_SEARCH_RESULT")
	}
	if err := v.Timeout("CACHE_TTL", cfg.CacheTTL); err != nil {
		errs.Add("CACHE_TTL", fieldMessage(err))
	}
	if err := v.RateLimit(cfg.RateLimit); err != nil {
		errs.Add("RATE_LIMIT", fieldMessage(err))
	}

	if err := errs.Err(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	return nil
}

func fieldMessage(err error) string {
	var ve validator.Error
	if errors.As(err, &ve) {
		return ve.Message
	}
	var se security.Error
	if errors.As(err, &se) {
		return se.Message
	}
	return err.Error()
}

func normalizeLogLevel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "info"
	}
	return value
}
