package logger

import (
	"context"
	"io"
	"os"
	"sync"
	"time"
)

// Fields is a structured key/value map attached to a log event.
type Fields map[string]any

// Logger is the injectable logging contract.
type Logger interface {
	Debug(msg string, fields ...Fields)
	Info(msg string, fields ...Fields)
	Warn(msg string, fields ...Fields)
	Error(msg string, fields ...Fields)
	Fatal(msg string, fields ...Fields)
	With(fields Fields) Logger
	WithCorrelationID(id string) Logger
	WithContext(ctx context.Context) Logger
}

// Entry is one log record.
type Entry struct {
	Timestamp     time.Time `json:"timestamp"`
	Level         string    `json:"level"`
	Message       string    `json:"message"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	Fields        Fields    `json:"fields,omitempty"`
}

// Sink writes a formatted entry.
type Sink interface {
	Write(entry Entry) error
	Close() error
}

// Options configures a base logger.
type Options struct {
	Level         Level
	CorrelationID string
	Fields        Fields
	Clock         func() time.Time
	FatalFunc     func(int) // defaults to os.Exit
}

// Base is a thread-safe logger writing to one or more sinks.
type Base struct {
	mu            sync.RWMutex
	level         Level
	sinks         []Sink
	correlationID string
	fields        Fields
	clock         func() time.Time
	fatalFunc     func(int)
}

// New constructs a Base logger with the given sinks.
func New(opts Options, sinks ...Sink) *Base {
	if opts.Clock == nil {
		opts.Clock = func() time.Time { return time.Now().UTC() }
	}
	if opts.FatalFunc == nil {
		opts.FatalFunc = os.Exit
	}
	fields := cloneFields(opts.Fields)
	out := make([]Sink, 0, len(sinks))
	for _, s := range sinks {
		if s != nil {
			out = append(out, s)
		}
	}
	return &Base{
		level:         opts.Level,
		sinks:         out,
		correlationID: opts.CorrelationID,
		fields:        fields,
		clock:         opts.Clock,
		fatalFunc:     opts.FatalFunc,
	}
}

func (l *Base) Debug(msg string, fields ...Fields) { l.log(LevelDebug, msg, fields...) }
func (l *Base) Info(msg string, fields ...Fields)  { l.log(LevelInfo, msg, fields...) }
func (l *Base) Warn(msg string, fields ...Fields)  { l.log(LevelWarn, msg, fields...) }
func (l *Base) Error(msg string, fields ...Fields) { l.log(LevelError, msg, fields...) }

func (l *Base) Fatal(msg string, fields ...Fields) {
	l.log(LevelFatal, msg, fields...)
	l.mu.RLock()
	fn := l.fatalFunc
	l.mu.RUnlock()
	if fn != nil {
		fn(1)
	}
}

// With returns a child logger with additional fields.
func (l *Base) With(fields Fields) Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()
	child := &Base{
		level:         l.level,
		sinks:         l.sinks,
		correlationID: l.correlationID,
		fields:        mergeFields(l.fields, fields),
		clock:         l.clock,
		fatalFunc:     l.fatalFunc,
	}
	return child
}

// WithCorrelationID returns a child logger bound to a correlation ID.
func (l *Base) WithCorrelationID(id string) Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()
	child := &Base{
		level:         l.level,
		sinks:         l.sinks,
		correlationID: id,
		fields:        cloneFields(l.fields),
		clock:         l.clock,
		fatalFunc:     l.fatalFunc,
	}
	return child
}

// WithContext extracts a correlation ID from context when present.
func (l *Base) WithContext(ctx context.Context) Logger {
	if ctx == nil {
		return l
	}
	if id, ok := CorrelationIDFromContext(ctx); ok && id != "" {
		return l.WithCorrelationID(id)
	}
	return l
}

// SetLevel updates the minimum level.
func (l *Base) SetLevel(level Level) {
	l.mu.Lock()
	l.level = level
	l.mu.Unlock()
}

// Close closes all sinks.
func (l *Base) Close() error {
	l.mu.RLock()
	sinks := append([]Sink(nil), l.sinks...)
	l.mu.RUnlock()
	var first error
	for _, s := range sinks {
		if err := s.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (l *Base) log(level Level, msg string, fieldSets ...Fields) {
	l.mu.RLock()
	min := l.level
	if !level.enabled(min) {
		l.mu.RUnlock()
		return
	}
	entry := Entry{
		Timestamp:     l.clock(),
		Level:         level.String(),
		Message:       msg,
		CorrelationID: l.correlationID,
		Fields:        mergeFields(l.fields, fieldSets...),
	}
	sinks := append([]Sink(nil), l.sinks...)
	l.mu.RUnlock()

	for _, s := range sinks {
		_ = s.Write(entry)
	}
}

func cloneFields(in Fields) Fields {
	if len(in) == 0 {
		return Fields{}
	}
	out := make(Fields, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mergeFields(base Fields, sets ...Fields) Fields {
	out := cloneFields(base)
	for _, set := range sets {
		for k, v := range set {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// Nop returns a logger that discards all output (useful in tests).
func Nop() Logger {
	return New(Options{
		Level:     LevelFatal + 1,
		FatalFunc: func(int) {},
	}, nopSink{})
}

type nopSink struct{}

func (nopSink) Write(Entry) error { return nil }
func (nopSink) Close() error      { return nil }

// writerSink adapts an io.Writer.
type writerSink struct {
	mu     sync.Mutex
	w      io.Writer
	format func(Entry) []byte
}

func (s *writerSink) Write(entry Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.w.Write(s.format(entry))
	return err
}

func (s *writerSink) Close() error {
	if c, ok := s.w.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
