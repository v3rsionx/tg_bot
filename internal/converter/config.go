package converter

import (
	"fmt"
	"time"
)

const (
	defaultBufferBytes   = 1 << 20 // 1 MiB
	defaultProgressEvery = 500 * time.Millisecond
	defaultDryRunRows    = 100
	defaultLogPath       = "logs/converter.log"
	defaultCheckpointDir = "data/converter"
)

// Config controls converter behavior.
type Config struct {
	// Sources are explicit CSV/TXT file paths.
	Sources []string
	// DryRun enables detection-only mode (first DryRunRows rows).
	DryRun bool
	// DryRunRows limits rows inspected in dry-run mode.
	DryRunRows int
	// Resume continues from a checkpoint when present.
	Resume bool
	// CheckpointDir stores per-file resume metadata.
	CheckpointDir string
	// LogPath is where skipped-row lines are appended.
	LogPath string
	// ProgressInterval controls progress callback frequency.
	ProgressInterval time.Duration
	// ReadBufferBytes sizes the buffered reader.
	ReadBufferBytes int
	// ForceDelimiter overrides autodetection when non-zero.
	ForceDelimiter rune
	// ForceEncoding overrides autodetection when non-empty.
	ForceEncoding EncodingName
}

// Validate checks configuration.
func (c Config) Validate() error {
	if len(c.Sources) == 0 {
		return fmt.Errorf("converter: Sources is required")
	}
	for i, s := range c.Sources {
		if s == "" {
			return fmt.Errorf("converter: Sources[%d] is empty", i)
		}
	}
	if c.DryRunRows < 0 {
		return fmt.Errorf("converter: DryRunRows must be >= 0")
	}
	if c.ReadBufferBytes < 0 {
		return fmt.Errorf("converter: ReadBufferBytes must be >= 0")
	}
	return nil
}

func (c Config) withDefaults() Config {
	if c.DryRunRows == 0 {
		c.DryRunRows = defaultDryRunRows
	}
	if c.ProgressInterval == 0 {
		c.ProgressInterval = defaultProgressEvery
	}
	if c.ReadBufferBytes == 0 {
		c.ReadBufferBytes = defaultBufferBytes
	}
	if c.LogPath == "" {
		c.LogPath = defaultLogPath
	}
	if c.CheckpointDir == "" {
		c.CheckpointDir = defaultCheckpointDir
	}
	return c
}
