package logger

import (
	"io"
	"os"
)

// ConsoleSink writes human-readable lines to an io.Writer (default stdout).
type ConsoleSink struct {
	*writerSink
}

// NewConsoleSink constructs a console sink.
func NewConsoleSink(w io.Writer) *ConsoleSink {
	if w == nil {
		w = os.Stdout
	}
	return &ConsoleSink{writerSink: &writerSink{w: w, format: formatText}}
}

// NewConsoleLogger constructs a text console logger.
func NewConsoleLogger(level Level) *Base {
	return New(Options{Level: level}, NewConsoleSink(os.Stdout))
}
