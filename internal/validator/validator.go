package validator

import "time"

// Validator is the injectable validation contract.
type Validator interface {
	TelegramToken(token string) error
	OwnerIDs(ids []int64) error
	OwnerIDsCSV(value string) ([]int64, error)
	SQLitePath(path string) error
	LMDBPath(field, path string) error
	DistinctPaths(paths map[string]string) error
	WorkerCount(value int) error
	BatchSize(value int) error
	Timeout(field string, value time.Duration) error
	PositiveInt(field string, value int) error
	NonNegativeInt(field string, value int) error
	MaxResults(value int) error
	RateLimit(value int) error
	LogLevel(value string) error
	Phone(value string) error
	Username(value string) error
	TelegramUserID(id int64) error
	SearchID(value string) error
	Command(value string) error
	EnvValue(key, value string) error
}

// Standard is the default dependency-injected validator implementation.
type Standard struct{}

// New constructs a Standard validator.
func New() *Standard {
	return &Standard{}
}

var _ Validator = (*Standard)(nil)
