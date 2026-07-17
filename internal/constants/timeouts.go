package constants

import "time"

// Timeout and duration defaults.
const (
	DefaultSearchTimeout = 5 * time.Second
	DefaultCacheTTL      = 5 * time.Minute
	DefaultHTTPTimeout   = 30 * time.Second
	DefaultShutdownWait  = 15 * time.Second
	DefaultWatchInterval = 2 * time.Second
	DefaultMetricsWindow = 24 * time.Hour
)

// Log rotation defaults.
const (
	DefaultLogMaxSizeMB  = 100
	DefaultLogMaxBackups = 14
	DefaultLogMaxAgeDays = 30
)
