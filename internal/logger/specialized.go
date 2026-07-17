package logger

import (
	"time"

	"github.com/v3rsi/tgbot-versionx/internal/constants"
)

// ContextLogger is a logger bound to a component name.
type ContextLogger struct {
	Logger
	component string
}

// NewContextLogger wraps base with a component field.
func NewContextLogger(base Logger, component string) *ContextLogger {
	if base == nil {
		base = Nop()
	}
	return &ContextLogger{
		Logger:    base.With(Fields{constants.FieldComponent: component}),
		component: component,
	}
}

// Component returns the bound component name.
func (l *ContextLogger) Component() string { return l.component }

// SearchLogger logs search-domain events.
type SearchLogger struct{ *ContextLogger }

// NewSearchLogger constructs a search logger.
func NewSearchLogger(base Logger) *SearchLogger {
	return &SearchLogger{NewContextLogger(base, "search")}
}

// Query logs a search query event.
func (l *SearchLogger) Query(userID int64, queryType, query string, found bool, latency time.Duration) {
	l.Info("search.query", Fields{
		constants.FieldUserID: userID,
		"type":                queryType,
		"query":               query,
		"found":               found,
		"latency_ms":          latency.Milliseconds(),
	})
}

// AdminLogger logs admin-domain events.
type AdminLogger struct{ *ContextLogger }

// NewAdminLogger constructs an admin logger.
func NewAdminLogger(base Logger) *AdminLogger {
	return &AdminLogger{NewContextLogger(base, "admin")}
}

// Action logs an admin action.
func (l *AdminLogger) Action(actorID int64, action string, fields Fields) {
	merged := Fields{
		"actor_id": actorID,
		"action":   action,
	}
	for k, v := range fields {
		merged[k] = v
	}
	l.Info("admin.action", merged)
}

// PerformanceLogger logs latency and throughput style events.
type PerformanceLogger struct{ *ContextLogger }

// NewPerformanceLogger constructs a performance logger.
func NewPerformanceLogger(base Logger) *PerformanceLogger {
	return &PerformanceLogger{NewContextLogger(base, "performance")}
}

// Timing logs an operation duration.
func (l *PerformanceLogger) Timing(operation string, latency time.Duration, fields Fields) {
	merged := Fields{
		constants.FieldOperation: operation,
		"latency_ms":             latency.Milliseconds(),
	}
	for k, v := range fields {
		merged[k] = v
	}
	l.Info("performance.timing", merged)
}
