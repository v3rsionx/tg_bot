package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/v3rsi/tgbot-versionx/internal/security"
	"github.com/v3rsi/tgbot-versionx/internal/validator"
)

// valueSource resolves configuration keys from layered sources.
// Process environment variables always override .env file values.
type valueSource struct {
	envFile   map[string]string
	sanitizer security.Sanitizer
	validator validator.Validator
}

func (s *valueSource) lookup(key string) (string, bool) {
	if value, ok := os.LookupEnv(key); ok {
		return value, true
	}
	if s.envFile != nil {
		if value, ok := s.envFile[key]; ok {
			return value, true
		}
	}
	return "", false
}

func (s *valueSource) applyOverrides(cfg *Config) error {
	if err := s.applyString("CONFIG_VERSION", &cfg.Version); err != nil {
		return err
	}
	if err := s.applyString("BOT_TOKEN", &cfg.BotToken); err != nil {
		return err
	}
	if raw, ok := s.lookup("BOT_OWNER_IDS"); ok {
		if err := s.guard("BOT_OWNER_IDS", raw); err != nil {
			return err
		}
		ids, err := s.validator.OwnerIDsCSV(raw)
		if err != nil {
			return err
		}
		cfg.BotOwnerIDs = ids
	}
	if err := s.applyString("SQLITE_PATH", &cfg.SQLitePath); err != nil {
		return err
	}
	if err := s.applyString("LMDB_ID_PATH", &cfg.LMDBIDPath); err != nil {
		return err
	}
	if err := s.applyString("LMDB_PHONE_PATH", &cfg.LMDBPhonePath); err != nil {
		return err
	}
	if err := s.applyString("LMDB_USERNAME_PATH", &cfg.LMDBUsernamePath); err != nil {
		return err
	}
	if err := s.applyString("LOG_LEVEL", &cfg.LogLevel); err != nil {
		return err
	}
	if err := s.applyInt("WORKER_COUNT", &cfg.WorkerCount); err != nil {
		return err
	}
	if err := s.applyInt("BATCH_SIZE", &cfg.BatchSize); err != nil {
		return err
	}
	if err := s.applyDuration("SEARCH_TIMEOUT", &cfg.SearchTimeout); err != nil {
		return err
	}
	if err := s.applyInt("POINTS_PER_SEARCH", &cfg.PointsPerSearch); err != nil {
		return err
	}

	maxResultsSet := false
	if raw, ok := s.lookup("MAX_RESULTS"); ok {
		if err := s.guard("MAX_RESULTS", raw); err != nil {
			return err
		}
		n, err := parsePositiveIntValue("MAX_RESULTS", raw)
		if err != nil {
			return err
		}
		cfg.MaxResults = n
		cfg.MaxSearchResult = n
		maxResultsSet = true
	}
	if raw, ok := s.lookup("MAX_SEARCH_RESULT"); ok {
		if err := s.guard("MAX_SEARCH_RESULT", raw); err != nil {
			return err
		}
		n, err := parsePositiveIntValue("MAX_SEARCH_RESULT", raw)
		if err != nil {
			return err
		}
		cfg.MaxSearchResult = n
		if !maxResultsSet {
			cfg.MaxResults = n
		}
	}

	if err := s.applyDuration("CACHE_TTL", &cfg.CacheTTL); err != nil {
		return err
	}
	if err := s.applyInt("RATE_LIMIT", &cfg.RateLimit); err != nil {
		return err
	}
	return nil
}

func (s *valueSource) applyString(key string, dest *string) error {
	raw, ok := s.lookup(key)
	if !ok {
		return nil
	}
	if err := s.guard(key, raw); err != nil {
		return err
	}
	*dest = strings.TrimSpace(raw)
	return nil
}

func (s *valueSource) applyInt(key string, dest *int) error {
	raw, ok := s.lookup(key)
	if !ok {
		return nil
	}
	if err := s.guard(key, raw); err != nil {
		return err
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return validator.Error{Field: key, Message: "must be an integer"}
	}
	*dest = n
	return nil
}

func (s *valueSource) applyDuration(key string, dest *time.Duration) error {
	raw, ok := s.lookup(key)
	if !ok {
		return nil
	}
	if err := s.guard(key, raw); err != nil {
		return err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return validator.Error{Field: key, Message: "must not be empty"}
	}
	// Allow bare integer seconds for operational convenience.
	if _, err := strconv.Atoi(raw); err == nil {
		raw += "s"
	}
	dur, err := time.ParseDuration(raw)
	if err != nil {
		return validator.Error{Field: key, Message: fmt.Sprintf("invalid duration %q", raw)}
	}
	*dest = dur
	return nil
}

func (s *valueSource) guard(key, value string) error {
	if s.validator != nil {
		if err := s.validator.EnvValue(key, value); err != nil {
			return err
		}
	}
	if s.sanitizer != nil {
		if err := s.sanitizer.PreventConfigInjection(key, value); err != nil {
			return err
		}
	}
	return nil
}

func parsePositiveIntValue(field, raw string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return 0, validator.Error{Field: field, Message: "must be a positive integer"}
	}
	return n, nil
}
