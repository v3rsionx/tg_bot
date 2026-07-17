package constants

// Rate limiting defaults.
const (
	DefaultRateLimit       = 5
	DefaultRateLimitWindow = "1m"
	OwnerRateLimitUnlimited = 0
)

// Points defaults.
const (
	DefaultPointsPerSearch = 1
	DefaultStartingPoints  = 0
)

// Cache defaults.
const (
	DefaultCacheTTLSeconds = 300
	DefaultCacheMaxEntries = 10_000
	DefaultSearchCacheSize = 5_000
	DefaultUserCacheSize   = 2_000
	DefaultAdminCacheSize  = 256
)

// Worker / batch defaults.
const (
	DefaultWorkerCount = 4
	DefaultBatchSize   = 1000
	MaxWorkerCount     = 10_000
	MaxBatchSize       = 1_000_000
)

// Input size limits.
const (
	MaxMessageBytes   = 4096
	MaxLMDBKeyBytes   = 511
	MaxPathBytes      = 4096
	MaxUsernameLength = 32
	MinUsernameLength = 3
	MaxPhoneDigits    = 15
	MinPhoneDigits    = 7
	MaxSearchIDLength = 32
)
