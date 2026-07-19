package importer

import (
	"fmt"
	"runtime"
	"time"
)

const (
	defaultDelimiter        = ','
	defaultWorkers          = 0 // resolved to GOMAXPROCS
	defaultBatchSize        = 4096
	defaultQueueSize        = 8192
	defaultProgressEvery    = 2 * time.Second
	defaultMaxLineBytes     = 16 * 1024 * 1024
	defaultIDColumn         = 0
	defaultPhoneColumn      = 1
	defaultUsernameColumn   = 2
	defaultCheckpointFile   = "data/importer.checkpoint.json"
	defaultReadBufferBytes  = 1 * 1024 * 1024
)

// Config controls streaming import behavior.
type Config struct {
	// Sources are absolute or relative paths to CSV/TXT files.
	Sources []string
	// Delimiter separates fields in each record.
	Delimiter rune
	// HasHeader skips the first non-empty line of each file when true.
	// When AutoMapHeaders detects a header, that line is skipped even if
	// HasHeader is false.
	HasHeader bool
	// AutoMapHeaders enables header-based column mapping for standard
	// (id,name,phone,username,extras) and legacy (id,phone,username) layouts.
	// When detection fails, fixed column indexes are used.
	AutoMapHeaders bool
	// Workers is the parse/validate worker count. Zero selects GOMAXPROCS.
	Workers int
	// BatchSize is the number of records per LMDB batch write.
	BatchSize int
	// QueueSize bounds in-flight parsed records between workers and writer.
	QueueSize int
	// CheckpointPath stores resume offsets. Empty disables resume persistence.
	CheckpointPath string
	// Resume enables seeking to the last successful checkpoint.
	Resume bool
	// SkipDuplicateIDs skips records whose ID already exists in id.lmdb.
	SkipDuplicateIDs bool
	// UpdateExisting updates indexes when an ID already exists.
	// Ignored when SkipDuplicateIDs is true.
	UpdateExisting bool
	// ProgressInterval controls progress callback frequency.
	ProgressInterval time.Duration
	// MaxLineBytes is the maximum accepted source line size.
	MaxLineBytes int
	// ReadBufferBytes is the buffered reader size used for streaming.
	ReadBufferBytes int
	// IDColumn is the zero-based source column for Telegram/user IDs.
	IDColumn int
	// PhoneColumn is the zero-based source column for phone numbers.
	PhoneColumn int
	// UsernameColumn is the zero-based source column for usernames.
	UsernameColumn int
	// NameColumn is the zero-based source column for display name.
	// Use -1 when absent (default).
	NameColumn int
	// ExtrasColumn is the zero-based source column for JSON extras.
	// Use -1 when absent (default).
	ExtrasColumn int

	// autoMapHeadersSet tracks whether AutoMapHeaders was explicitly configured.
	autoMapHeadersSet bool
}

// Validate checks importer configuration for production use.
func (c Config) Validate() error {
	if len(c.Sources) == 0 {
		return fmt.Errorf("importer: Sources is required")
	}
	for i, source := range c.Sources {
		if source == "" {
			return fmt.Errorf("importer: Sources[%d] is empty", i)
		}
	}
	if c.Delimiter == 0 {
		return fmt.Errorf("importer: Delimiter is required")
	}
	if c.Workers < 0 {
		return fmt.Errorf("importer: Workers must be >= 0")
	}
	if c.BatchSize < 0 {
		return fmt.Errorf("importer: BatchSize must be >= 0")
	}
	if c.QueueSize < 0 {
		return fmt.Errorf("importer: QueueSize must be >= 0")
	}
	if c.MaxLineBytes < 0 {
		return fmt.Errorf("importer: MaxLineBytes must be >= 0")
	}
	if c.ReadBufferBytes < 0 {
		return fmt.Errorf("importer: ReadBufferBytes must be >= 0")
	}
	if c.IDColumn < 0 || c.PhoneColumn < 0 || c.UsernameColumn < 0 {
		return fmt.Errorf("importer: column indexes must be >= 0")
	}
	if c.NameColumn < unsetColumn {
		return fmt.Errorf("importer: NameColumn must be >= -1")
	}
	if c.ExtrasColumn < unsetColumn {
		return fmt.Errorf("importer: ExtrasColumn must be >= -1")
	}
	if c.SkipDuplicateIDs && c.UpdateExisting {
		return fmt.Errorf("importer: SkipDuplicateIDs and UpdateExisting are mutually exclusive")
	}
	return nil
}

// WithAutoMapHeaders returns a copy with AutoMapHeaders explicitly set.
func (c Config) WithAutoMapHeaders(enabled bool) Config {
	c.AutoMapHeaders = enabled
	c.autoMapHeadersSet = true
	return c
}

// withDefaults returns a copy of Config with production defaults applied.
func (c Config) withDefaults() Config {
	if c.Delimiter == 0 {
		c.Delimiter = defaultDelimiter
	}
	if c.Workers == 0 {
		c.Workers = runtime.GOMAXPROCS(0)
		if c.Workers < 1 {
			c.Workers = 1
		}
	}
	if c.BatchSize == 0 {
		c.BatchSize = defaultBatchSize
	}
	if c.QueueSize == 0 {
		c.QueueSize = defaultQueueSize
	}
	if c.ProgressInterval == 0 {
		c.ProgressInterval = defaultProgressEvery
	}
	if c.MaxLineBytes == 0 {
		c.MaxLineBytes = defaultMaxLineBytes
	}
	if c.ReadBufferBytes == 0 {
		c.ReadBufferBytes = defaultReadBufferBytes
	}
	if c.CheckpointPath == "" && c.Resume {
		c.CheckpointPath = defaultCheckpointFile
	}
	// Default column layout: id,phone,username (legacy positional fallback).
	if c.IDColumn == 0 && c.PhoneColumn == 0 && c.UsernameColumn == 0 {
		c.IDColumn = defaultIDColumn
		c.PhoneColumn = defaultPhoneColumn
		c.UsernameColumn = defaultUsernameColumn
	}
	if !c.autoMapHeadersSet {
		c.AutoMapHeaders = true
	}
	// Optional columns default to unset (-1). Zero is treated as unset unless the
	// caller disabled AutoMapHeaders and set positional indexes explicitly via
	// NameColumn/ExtrasColumn together with non-default phone/username layout.
	if c.NameColumn == 0 && c.ExtrasColumn == 0 {
		c.NameColumn = unsetColumn
		c.ExtrasColumn = unsetColumn
	}
	if c.NameColumn < unsetColumn {
		c.NameColumn = unsetColumn
	}
	if c.ExtrasColumn < unsetColumn {
		c.ExtrasColumn = unsetColumn
	}
	// Skip duplicates by default unless explicit update mode is enabled.
	if c.UpdateExisting {
		c.SkipDuplicateIDs = false
	} else {
		c.SkipDuplicateIDs = true
	}
	return c
}
