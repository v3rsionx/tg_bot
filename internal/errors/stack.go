package errors

import (
	"fmt"
	"runtime"
	"strings"
)

const defaultStackDepth = 32

// StackFrame is one frame of an optional stack trace.
type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// CaptureStack records the current call stack, skipping skip frames.
func CaptureStack(skip int) []StackFrame {
	pcs := make([]uintptr, defaultStackDepth)
	n := runtime.Callers(skip+2, pcs)
	if n == 0 {
		return nil
	}
	frames := runtime.CallersFrames(pcs[:n])
	out := make([]StackFrame, 0, n)
	for {
		frame, more := frames.Next()
		out = append(out, StackFrame{
			Function: frame.Function,
			File:     frame.File,
			Line:     frame.Line,
		})
		if !more {
			break
		}
	}
	return out
}

func formatStack(frames []StackFrame) string {
	if len(frames) == 0 {
		return ""
	}
	var b strings.Builder
	for _, f := range frames {
		b.WriteString(fmt.Sprintf("%s\n\t%s:%d\n", f.Function, f.File, f.Line))
	}
	return b.String()
}
