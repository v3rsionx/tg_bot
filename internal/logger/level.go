package logger

import (
	"strings"

	"github.com/v3rsionx/tg_bot/internal/constants"
)

// Level is a log severity.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// String returns the canonical level name.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return constants.LogLevelDebug
	case LevelInfo:
		return constants.LogLevelInfo
	case LevelWarn:
		return constants.LogLevelWarn
	case LevelError:
		return constants.LogLevelError
	case LevelFatal:
		return constants.LogLevelFatal
	default:
		return constants.LogLevelInfo
	}
}

// ParseLevel parses a level name. Unknown values default to Info.
func ParseLevel(s string) Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case constants.LogLevelDebug:
		return LevelDebug
	case constants.LogLevelInfo:
		return LevelInfo
	case constants.LogLevelWarn, "warning":
		return LevelWarn
	case constants.LogLevelError:
		return LevelError
	case constants.LogLevelFatal:
		return LevelFatal
	default:
		return LevelInfo
	}
}

func (l Level) enabled(min Level) bool {
	return l >= min
}
