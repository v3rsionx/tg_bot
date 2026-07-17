package validator

import (
	"strings"
	"time"
)

const (
	maxWorkerCount = 10_000
	maxBatchSize   = 1_000_000
	maxMaxResults  = 10_000
	maxRateLimit   = 10_000
	maxTimeout     = 24 * time.Hour
)

// WorkerCount validates importer/search worker concurrency.
func (v *Standard) WorkerCount(value int) error {
	if value < 1 || value > maxWorkerCount {
		return Error{Field: "WORKER_COUNT", Message: "must be between 1 and 10000"}
	}
	return nil
}

// BatchSize validates batch write sizing.
func (v *Standard) BatchSize(value int) error {
	if value < 1 || value > maxBatchSize {
		return Error{Field: "BATCH_SIZE", Message: "must be between 1 and 1000000"}
	}
	return nil
}

// Timeout validates a positive timeout duration.
func (v *Standard) Timeout(field string, value time.Duration) error {
	if value <= 0 {
		return Error{Field: field, Message: "must be greater than zero"}
	}
	if value > maxTimeout {
		return Error{Field: field, Message: "must be <= 24h"}
	}
	return nil
}

// PositiveInt validates a positive integer environment value.
func (v *Standard) PositiveInt(field string, value int) error {
	if value <= 0 {
		return Error{Field: field, Message: "must be a positive integer"}
	}
	return nil
}

// NonNegativeInt validates a non-negative integer environment value.
func (v *Standard) NonNegativeInt(field string, value int) error {
	if value < 0 {
		return Error{Field: field, Message: "must be >= 0"}
	}
	return nil
}

// MaxResults validates MAX_RESULTS / MAX_SEARCH_RESULT.
func (v *Standard) MaxResults(value int) error {
	if value < 1 || value > maxMaxResults {
		return Error{Field: "MAX_RESULTS", Message: "must be between 1 and 10000"}
	}
	return nil
}

// RateLimit validates RATE_LIMIT (requests per window).
func (v *Standard) RateLimit(value int) error {
	if value < 1 || value > maxRateLimit {
		return Error{Field: "RATE_LIMIT", Message: "must be between 1 and 10000"}
	}
	return nil
}

// LogLevel validates a logging level.
func (v *Standard) LogLevel(value string) error {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug", "info", "warn", "error":
		return nil
	default:
		return Error{Field: "LOG_LEVEL", Message: "must be one of: debug, info, warn, error"}
	}
}
