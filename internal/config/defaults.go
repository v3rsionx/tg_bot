package config

import "time"

// CurrentConfigVersion is the configuration schema version produced by defaults.
const CurrentConfigVersion = "1"

const (
	defaultEnvFile  = ".env"
	defaultJSONFile = "configs/config.json"

	maxJSONBytes     = 1 << 20 // 1 MiB
	maxEnvFileBytes  = 1 << 20 // 1 MiB
	maxEnvFileLines  = 10_000
	maxEnvLineBytes  = 64 << 10 // 64 KiB
	defaultWatchInterval = 2 * time.Second
)

// SupportedVersions lists accepted CONFIG_VERSION values.
var SupportedVersions = map[string]struct{}{
	CurrentConfigVersion: {},
}

// Defaults returns a Config populated with safe production defaults.
// Required secrets and paths remain empty until supplied by JSON, .env, or env.
func Defaults() Config {
	return Config{
		Version:         CurrentConfigVersion,
		LogLevel:        "info",
		WorkerCount:     4,
		BatchSize:       1000,
		SearchTimeout:   5 * time.Second,
		PointsPerSearch: 1,
		MaxResults:      100,
		MaxSearchResult: 100,
		CacheTTL:        5 * time.Minute,
		RateLimit:       5,
	}
}
