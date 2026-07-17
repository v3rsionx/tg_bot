package constants

// Configuration keys.
const (
	ConfigVersion        = "CONFIG_VERSION"
	ConfigBotToken       = "BOT_TOKEN"
	ConfigBotOwnerIDs    = "BOT_OWNER_IDS"
	ConfigSQLitePath     = "SQLITE_PATH"
	ConfigLMDBIDPath     = "LMDB_ID_PATH"
	ConfigLMDBPhonePath  = "LMDB_PHONE_PATH"
	ConfigLMDBUsernamePath = "LMDB_USERNAME_PATH"
	ConfigLogLevel       = "LOG_LEVEL"
	ConfigWorkerCount    = "WORKER_COUNT"
	ConfigBatchSize      = "BATCH_SIZE"
	ConfigSearchTimeout  = "SEARCH_TIMEOUT"
	ConfigPointsPerSearch = "POINTS_PER_SEARCH"
	ConfigMaxResults     = "MAX_RESULTS"
	ConfigMaxSearchResult = "MAX_SEARCH_RESULT"
	ConfigCacheTTL       = "CACHE_TTL"
	ConfigRateLimit      = "RATE_LIMIT"
)

// Configuration file defaults.
const (
	DefaultEnvFile  = ".env"
	DefaultJSONFile = "configs/config.json"
	ConfigSchemaVersion = "1"
)
