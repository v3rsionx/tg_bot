package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// jsonFile is the on-disk JSON configuration document.
type jsonFile struct {
	Version          string  `json:"version"`
	BotToken         string  `json:"bot_token"`
	BotOwnerIDs      []int64 `json:"bot_owner_ids"`
	SQLitePath       string  `json:"sqlite_path"`
	LMDBIDPath       string  `json:"lmdb_id_path"`
	LMDBPhonePath    string  `json:"lmdb_phone_path"`
	LMDBUsernamePath string  `json:"lmdb_username_path"`
	LogLevel         string  `json:"log_level"`
	WorkerCount      *int    `json:"worker_count"`
	BatchSize        *int    `json:"batch_size"`
	SearchTimeout    string  `json:"search_timeout"`
	PointsPerSearch  *int    `json:"points_per_search"`
	MaxResults       *int    `json:"max_results"`
	MaxSearchResult  *int    `json:"max_search_result"`
	CacheTTL         string  `json:"cache_ttl"`
	RateLimit        *int    `json:"rate_limit"`
}

func readJSONFile(path string) (*jsonFile, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory", path)
	}
	if info.Size() > maxJSONBytes {
		return nil, fmt.Errorf("%s exceeds maximum size of %d bytes", path, maxJSONBytes)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("%s is empty", path)
	}

	var doc jsonFile
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &doc, nil
}

func (d *jsonFile) apply(cfg *Config) error {
	if d == nil {
		return nil
	}
	if d.Version != "" {
		cfg.Version = d.Version
	}
	if d.BotToken != "" {
		cfg.BotToken = d.BotToken
	}
	if len(d.BotOwnerIDs) > 0 {
		cfg.BotOwnerIDs = append([]int64(nil), d.BotOwnerIDs...)
	}
	if d.SQLitePath != "" {
		cfg.SQLitePath = d.SQLitePath
	}
	if d.LMDBIDPath != "" {
		cfg.LMDBIDPath = d.LMDBIDPath
	}
	if d.LMDBPhonePath != "" {
		cfg.LMDBPhonePath = d.LMDBPhonePath
	}
	if d.LMDBUsernamePath != "" {
		cfg.LMDBUsernamePath = d.LMDBUsernamePath
	}
	if d.LogLevel != "" {
		cfg.LogLevel = d.LogLevel
	}
	if d.WorkerCount != nil {
		cfg.WorkerCount = *d.WorkerCount
	}
	if d.BatchSize != nil {
		cfg.BatchSize = *d.BatchSize
	}
	if d.SearchTimeout != "" {
		dur, err := time.ParseDuration(d.SearchTimeout)
		if err != nil {
			return fmt.Errorf("search_timeout: invalid duration %q", d.SearchTimeout)
		}
		cfg.SearchTimeout = dur
	}
	if d.PointsPerSearch != nil {
		cfg.PointsPerSearch = *d.PointsPerSearch
	}
	if d.MaxResults != nil {
		cfg.MaxResults = *d.MaxResults
		cfg.MaxSearchResult = *d.MaxResults
	}
	if d.MaxSearchResult != nil {
		cfg.MaxSearchResult = *d.MaxSearchResult
		if d.MaxResults == nil {
			cfg.MaxResults = *d.MaxSearchResult
		}
	}
	if d.CacheTTL != "" {
		dur, err := time.ParseDuration(d.CacheTTL)
		if err != nil {
			return fmt.Errorf("cache_ttl: invalid duration %q", d.CacheTTL)
		}
		cfg.CacheTTL = dur
	}
	if d.RateLimit != nil {
		cfg.RateLimit = *d.RateLimit
	}
	return nil
}
