package logger

// FileSink writes text logs to a rotating file.
type FileSink struct {
	*writerSink
	rotator *RotatingFile
}

// NewFileSink constructs a rotating file sink.
func NewFileSink(opts RotateOptions) (*FileSink, error) {
	rot, err := NewRotatingFile(opts)
	if err != nil {
		return nil, err
	}
	return &FileSink{
		writerSink: &writerSink{w: rot, format: formatText},
		rotator:    rot,
	}, nil
}

// NewJSONFileSink constructs a rotating JSON file sink.
func NewJSONFileSink(opts RotateOptions) (*FileSink, error) {
	rot, err := NewRotatingFile(opts)
	if err != nil {
		return nil, err
	}
	return &FileSink{
		writerSink: &writerSink{w: rot, format: formatJSON},
		rotator:    rot,
	}, nil
}

// Close closes the underlying rotator.
func (s *FileSink) Close() error {
	if s == nil || s.rotator == nil {
		return nil
	}
	return s.rotator.Close()
}

// NewFileLogger constructs a text file logger with rotation.
func NewFileLogger(level Level, opts RotateOptions) (*Base, error) {
	sink, err := NewFileSink(opts)
	if err != nil {
		return nil, err
	}
	return New(Options{Level: level}, sink), nil
}

// NewMultiLogger writes to all provided sinks.
func NewMultiLogger(level Level, sinks ...Sink) *Base {
	return New(Options{Level: level}, sinks...)
}
