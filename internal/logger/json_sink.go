package logger

import (
	"io"
	"os"
)

// JSONSink writes one JSON object per line.
type JSONSink struct {
	*writerSink
}

// NewJSONSink constructs a JSON sink.
func NewJSONSink(w io.Writer) *JSONSink {
	if w == nil {
		w = os.Stdout
	}
	return &JSONSink{writerSink: &writerSink{w: w, format: formatJSON}}
}

// NewJSONLogger constructs a JSON logger writing to w.
func NewJSONLogger(level Level, w io.Writer) *Base {
	return New(Options{Level: level}, NewJSONSink(w))
}
