package importer

import (
	"log"
	"os"
)

// Logger is the injectable logging contract used by the importer.
type Logger interface {
	// Debugf logs a debug-level message.
	Debugf(format string, args ...any)
	// Infof logs an info-level message.
	Infof(format string, args ...any)
	// Warnf logs a warning-level message.
	Warnf(format string, args ...any)
	// Errorf logs an error-level message.
	Errorf(format string, args ...any)
}

// StdLogger adapts the standard library logger to Logger.
type StdLogger struct {
	log *log.Logger
}

// NewStdLogger constructs a StdLogger writing to stderr.
func NewStdLogger() *StdLogger {
	return &StdLogger{log: log.New(os.Stderr, "importer: ", log.LstdFlags|log.Lmsgprefix)}
}

// Debugf logs a debug-level message.
func (l *StdLogger) Debugf(format string, args ...any) {
	l.log.Printf("DEBUG "+format, args...)
}

// Infof logs an info-level message.
func (l *StdLogger) Infof(format string, args ...any) {
	l.log.Printf("INFO "+format, args...)
}

// Warnf logs a warning-level message.
func (l *StdLogger) Warnf(format string, args ...any) {
	l.log.Printf("WARN "+format, args...)
}

// Errorf logs an error-level message.
func (l *StdLogger) Errorf(format string, args ...any) {
	l.log.Printf("ERROR "+format, args...)
}

// NopLogger discards all log output.
type NopLogger struct{}

// Debugf discards the message.
func (NopLogger) Debugf(format string, args ...any) {}

// Infof discards the message.
func (NopLogger) Infof(format string, args ...any) {}

// Warnf discards the message.
func (NopLogger) Warnf(format string, args ...any) {}

// Errorf discards the message.
func (NopLogger) Errorf(format string, args ...any) {}
