package constants

// Logging defaults.
const (
	DefaultLogLevel     = "info"
	DefaultLogFileName  = "logs/app.log"
	LogLevelDebug       = "debug"
	LogLevelInfo        = "info"
	LogLevelWarn        = "warn"
	LogLevelError       = "error"
	LogLevelFatal       = "fatal"
)

// Metrics / runtime sampling.
const (
	MetricsNamespace = "tgbot"
)

// Correlation / context keys (string form for logger fields).
const (
	FieldCorrelationID = "correlation_id"
	FieldOperation     = "operation"
	FieldUserID        = "user_id"
	FieldComponent     = "component"
)
