package importer

import (
	"time"

	"github.com/v3rsionx/tg_bot/internal/database/lmdb"
)

// Stores holds destination exact-lookup LMDB engines.
type Stores struct {
	// ID stores id -> payload lookups.
	ID lmdb.Engine
	// Phone stores phone -> id lookups.
	Phone lmdb.Engine
	// Username stores username -> id lookups.
	Username lmdb.Engine
}

// Validate ensures all destination engines are provided.
func (s Stores) Validate() error {
	if s.ID == nil {
		return errStores("ID")
	}
	if s.Phone == nil {
		return errStores("Phone")
	}
	if s.Username == nil {
		return errStores("Username")
	}
	return nil
}

// Record is a validated source row ready for indexing.
type Record struct {
	ID       string
	Phone    string
	Username string
	File     string
	Line     uint64
	Offset   int64
}

// Statistics captures cumulative import counters and speed metrics.
type Statistics struct {
	FilesTotal       int
	FilesCompleted   int
	BytesRead        int64
	BytesTotal       int64
	LinesRead        uint64
	RecordsParsed    uint64
	RecordsInvalid   uint64
	Duplicates       uint64
	Inserts          uint64
	Updates          uint64
	PhoneWrites      uint64
	UsernameWrites   uint64
	BatchesWritten   uint64
	StartedAt        time.Time
	FinishedAt       time.Time
	RecordsPerSecond float64
	BytesPerSecond   float64
	ETA              time.Duration
}

// Progress is a point-in-time import progress snapshot.
type Progress struct {
	CurrentFile string
	Statistics  Statistics
}

// ProgressFunc receives periodic progress updates.
type ProgressFunc func(Progress)

// rawJob is an unparsed source line for worker processing.
type rawJob struct {
	File   string
	Line   uint64
	Offset int64
	Text   string
}
