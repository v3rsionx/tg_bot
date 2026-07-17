package search

import (
	"time"

	"github.com/v3rsi/tgbot-versionx/internal/database/lmdb"
)

// QueryType identifies which exact index was queried.
type QueryType string

const (
	// QueryTypeID searches the ID index directly.
	QueryTypeID QueryType = "id"
	// QueryTypePhone searches phone -> id -> record.
	QueryTypePhone QueryType = "phone"
	// QueryTypeUsername searches username -> id -> record.
	QueryTypeUsername QueryType = "username"
)

// Record is the strongly typed full record returned by exact lookups.
type Record struct {
	ID       string
	Phone    string
	Username string
}

// Result contains a lookup outcome with timing and cache metadata.
type Result struct {
	Record    Record
	Found     bool
	CacheHit  bool
	QueryType QueryType
	Query     string
	Latency   time.Duration
}

// Statistics captures cumulative search-engine metrics.
type Statistics struct {
	Queries         uint64
	Hits            uint64
	Misses          uint64
	CacheHits       uint64
	CacheMisses     uint64
	InvalidQueries  uint64
	Errors          uint64
	IDQueries       uint64
	PhoneQueries    uint64
	UsernameQueries uint64
	TotalLatency    time.Duration
	AverageLatency  time.Duration
}

// Stores holds injectable exact-lookup LMDB engines.
type Stores struct {
	// ID stores id -> payload.
	ID lmdb.Engine
	// Phone stores phone -> id.
	Phone lmdb.Engine
	// Username stores username -> id.
	Username lmdb.Engine
}

// Validate ensures all destination engines are provided.
func (s Stores) Validate() error {
	if s.ID == nil {
		return errStore("ID")
	}
	if s.Phone == nil {
		return errStore("Phone")
	}
	if s.Username == nil {
		return errStore("Username")
	}
	return nil
}
