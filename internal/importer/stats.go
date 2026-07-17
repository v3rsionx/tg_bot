package importer

import (
	"sync"
	"sync/atomic"
	"time"
)

// statsAccumulator tracks import counters in a concurrency-safe way.
type statsAccumulator struct {
	mu             sync.Mutex
	filesTotal     int
	filesCompleted int
	bytesTotal     int64
	startedAt      time.Time
	finishedAt     time.Time

	bytesRead      atomic.Int64
	linesRead      atomic.Uint64
	recordsParsed  atomic.Uint64
	recordsInvalid atomic.Uint64
	duplicates     atomic.Uint64
	inserts        atomic.Uint64
	updates        atomic.Uint64
	phoneWrites    atomic.Uint64
	usernameWrites atomic.Uint64
	batchesWritten atomic.Uint64
}

// newStatsAccumulator constructs counters for an import run.
func newStatsAccumulator(filesTotal int) *statsAccumulator {
	return &statsAccumulator{
		filesTotal: filesTotal,
		startedAt:  time.Now().UTC(),
	}
}

// addBytesTotal increases known total source size.
func (s *statsAccumulator) addBytesTotal(n int64) {
	s.mu.Lock()
	s.bytesTotal += n
	s.mu.Unlock()
}

// markFileCompleted increments completed file count.
func (s *statsAccumulator) markFileCompleted() {
	s.mu.Lock()
	s.filesCompleted++
	s.mu.Unlock()
}

// finish records the end timestamp.
func (s *statsAccumulator) finish() {
	s.mu.Lock()
	s.finishedAt = time.Now().UTC()
	s.mu.Unlock()
}

// snapshot returns a consistent Statistics view including speed and ETA.
func (s *statsAccumulator) snapshot(now time.Time) Statistics {
	s.mu.Lock()
	startedAt := s.startedAt
	finishedAt := s.finishedAt
	filesTotal := s.filesTotal
	filesCompleted := s.filesCompleted
	bytesTotal := s.bytesTotal
	s.mu.Unlock()

	if finishedAt.IsZero() {
		finishedAt = now
	}
	elapsed := finishedAt.Sub(startedAt)
	if elapsed <= 0 {
		elapsed = time.Nanosecond
	}

	bytesRead := s.bytesRead.Load()
	linesRead := s.linesRead.Load()
	inserts := s.inserts.Load()
	updates := s.updates.Load()
	processed := inserts + updates

	recordsPerSec := float64(processed) / elapsed.Seconds()
	bytesPerSec := float64(bytesRead) / elapsed.Seconds()

	var eta time.Duration
	if bytesTotal > 0 && bytesRead > 0 && bytesRead < bytesTotal && bytesPerSec > 0 {
		remaining := float64(bytesTotal-bytesRead) / bytesPerSec
		eta = time.Duration(remaining * float64(time.Second))
	}

	return Statistics{
		FilesTotal:       filesTotal,
		FilesCompleted:   filesCompleted,
		BytesRead:        bytesRead,
		BytesTotal:       bytesTotal,
		LinesRead:        linesRead,
		RecordsParsed:    s.recordsParsed.Load(),
		RecordsInvalid:   s.recordsInvalid.Load(),
		Duplicates:       s.duplicates.Load(),
		Inserts:          inserts,
		Updates:          updates,
		PhoneWrites:      s.phoneWrites.Load(),
		UsernameWrites:   s.usernameWrites.Load(),
		BatchesWritten:   s.batchesWritten.Load(),
		StartedAt:        startedAt,
		FinishedAt:       finishedAt,
		RecordsPerSecond: recordsPerSec,
		BytesPerSecond:   bytesPerSec,
		ETA:              eta,
	}
}
