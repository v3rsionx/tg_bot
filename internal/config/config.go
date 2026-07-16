// Package config loads and validates application configuration from the
// process environment and an optional .env file.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const defaultEnvFile = ".env"

var botTokenPattern = regexp.MustCompile(`^[0-9]{5,15}:[A-Za-z0-9_-]{20,}$`)

// Config contains all validated application configuration.
type Config struct {
	BotToken         string
	BotOwnerIDs      []int64
	SQLitePath       string
	LMDBIDPath       string
	LMDBPhonePath    string
	LMDBUsernamePath string
	LogLevel         string
	PointsPerSearch  int
	MaxSearchResult  int
}

// Load reads .env when present, applies environment-variable overrides, and
// returns a fully validated configuration.
func Load() (*Config, error) {
	if err := loadEnvFile(defaultEnvFile); err != nil {
		return nil, err
	}

	config, err := loadFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// MustLoad returns a fully validated configuration or panics. It is intended
// for executable entry points, where invalid configuration must stop startup.
func MustLoad() *Config {
	config, err := Load()
	if err != nil {
		panic(err)
	}

	return config
}

func loadFromEnvironment() (*Config, error) {
	botToken, err := requiredValue("BOT_TOKEN")
	if err != nil {
		return nil, err
	}
	if !botTokenPattern.MatchString(botToken) {
		return nil, errors.New("BOT_TOKEN must be a valid Telegram bot token")
	}

	ownerIDs, err := parseOwnerIDs(os.Getenv("BOT_OWNER_IDS"))
	if err != nil {
		return nil, err
	}

	sqlitePath, err := parseFilePath("SQLITE_PATH", os.Getenv("SQLITE_PATH"))
	if err != nil {
		return nil, err
	}
	lmdbIDPath, err := parseDirectoryPath("LMDB_ID_PATH", os.Getenv("LMDB_ID_PATH"))
	if err != nil {
		return nil, err
	}
	lmdbPhonePath, err := parseDirectoryPath("LMDB_PHONE_PATH", os.Getenv("LMDB_PHONE_PATH"))
	if err != nil {
		return nil, err
	}
	lmdbUsernamePath, err := parseDirectoryPath("LMDB_USERNAME_PATH", os.Getenv("LMDB_USERNAME_PATH"))
	if err != nil {
		return nil, err
	}
	if pathsContainDuplicates(lmdbIDPath, lmdbPhonePath, lmdbUsernamePath) {
		return nil, errors.New("LMDB_ID_PATH, LMDB_PHONE_PATH, and LMDB_USERNAME_PATH must be distinct")
	}

	logLevel, err := parseLogLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		return nil, err
	}
	pointsPerSearch, err := parsePositiveInt("POINTS_PER_SEARCH", os.Getenv("POINTS_PER_SEARCH"))
	if err != nil {
		return nil, err
	}
	maxSearchResult, err := parsePositiveInt("MAX_SEARCH_RESULT", os.Getenv("MAX_SEARCH_RESULT"))
	if err != nil {
		return nil, err
	}

	return &Config{
		BotToken:         botToken,
		BotOwnerIDs:      ownerIDs,
		SQLitePath:       sqlitePath,
		LMDBIDPath:       lmdbIDPath,
		LMDBPhonePath:    lmdbPhonePath,
		LMDBUsernamePath: lmdbUsernamePath,
		LogLevel:         logLevel,
		PointsPerSearch:  pointsPerSearch,
		MaxSearchResult:  maxSearchResult,
	}, nil
}

func requiredValue(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}

	return value, nil
}

func parseOwnerIDs(value string) ([]int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("BOT_OWNER_IDS is required")
	}

	parts := strings.Split(value, ",")
	ownerIDs := make([]int64, 0, len(parts))
	seen := make(map[int64]struct{}, len(parts))

	for _, part := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || id <= 0 {
			return nil, errors.New("BOT_OWNER_IDS must be a comma-separated list of positive integers")
		}
		if _, exists := seen[id]; exists {
			return nil, errors.New("BOT_OWNER_IDS must not contain duplicate IDs")
		}

		seen[id] = struct{}{}
		ownerIDs = append(ownerIDs, id)
	}

	return ownerIDs, nil
}

func parseFilePath(name, value string) (string, error) {
	value, err := requiredPath(name, value)
	if err != nil {
		return "", err
	}
	if filepath.Base(value) == "." {
		return "", fmt.Errorf("%s must identify a file", name)
	}

	return value, nil
}

func parseDirectoryPath(name, value string) (string, error) {
	return requiredPath(name, value)
}

func requiredPath(name, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}

	path := filepath.Clean(value)
	if path == "." {
		return "", fmt.Errorf("%s must not be the current directory", name)
	}

	return path, nil
}

func pathsContainDuplicates(paths ...string) bool {
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		normalized := filepath.Clean(path)
		if _, exists := seen[normalized]; exists {
			return true
		}
		seen[normalized] = struct{}{}
	}

	return false
}

func parseLogLevel(value string) (string, error) {
	switch level := strings.ToLower(strings.TrimSpace(value)); level {
	case "debug", "info", "warn", "error":
		return level, nil
	default:
		return "", errors.New("LOG_LEVEL must be one of: debug, info, warn, error")
	}
}

func parsePositiveInt(name, value string) (int, error) {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}

	return number, nil
}

func loadEnvFile(path string) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		key, value, ok, err := parseEnvLine(scanner.Text())
		if err != nil {
			return fmt.Errorf("parse %s line %d: %w", path, lineNumber, err)
		}
		if !ok {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("set %s from %s: %w", key, path, err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	return nil
}

func parseEnvLine(line string) (key, value string, ok bool, err error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false, nil
	}

	line = strings.TrimPrefix(line, "export ")
	key, value, found := strings.Cut(line, "=")
	if !found {
		return "", "", false, errors.New("expected KEY=VALUE")
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return "", "", false, errors.New("environment variable name is empty")
	}

	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value, err = strconv.Unquote(value)
		if err != nil {
			return "", "", false, fmt.Errorf("invalid quoted value: %w", err)
		}
	} else if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		value = value[1 : len(value)-1]
	}

	return key, value, true, nil
}
