package converter

import (
	"fmt"
	"sync/atomic"
	"time"
)

type statsAccumulator struct {
	inputRows   atomic.Uint64
	outputRows  atomic.Uint64
	skippedRows atomic.Uint64
	bytesRead   atomic.Int64
	bytesTotal  atomic.Int64
	startedAt   time.Time
	finishedAt  atomic.Value // time.Time
}

func newStats() *statsAccumulator {
	s := &statsAccumulator{startedAt: time.Now().UTC()}
	s.finishedAt.Store(time.Time{})
	return s
}

func (s *statsAccumulator) snapshot() Statistics {
	started := s.startedAt
	finished, _ := s.finishedAt.Load().(time.Time)
	end := finished
	if end.IsZero() {
		end = time.Now().UTC()
	}
	elapsed := end.Sub(started).Seconds()
	in := s.inputRows.Load()
	var rps float64
	if elapsed > 0 {
		rps = float64(in) / elapsed
	}
	return Statistics{
		InputRows:   in,
		OutputRows:  s.outputRows.Load(),
		SkippedRows: s.skippedRows.Load(),
		BytesRead:   s.bytesRead.Load(),
		BytesTotal:  s.bytesTotal.Load(),
		StartedAt:   started,
		FinishedAt:  finished,
		RowsPerSec:  rps,
	}
}

func (s *statsAccumulator) finish() {
	s.finishedAt.Store(time.Now().UTC())
}

func buildProgress(file string, st Statistics) Progress {
	var pct float64
	if st.BytesTotal > 0 {
		pct = float64(st.BytesRead) / float64(st.BytesTotal) * 100
		if pct > 100 {
			pct = 100
		}
	}
	var eta time.Duration
	if st.RowsPerSec > 0 && st.BytesTotal > st.BytesRead && st.BytesRead > 0 {
		remain := float64(st.BytesTotal-st.BytesRead) / (float64(st.BytesRead) / time.Since(st.StartedAt).Seconds())
		eta = time.Duration(remain * float64(time.Second))
	}
	return Progress{
		File:       file,
		Processed:  st.InputRows,
		Output:     st.OutputRows,
		Skipped:    st.SkippedRows,
		Percent:    pct,
		RowsPerSec: st.RowsPerSec,
		ETA:        eta,
		BytesRead:  st.BytesRead,
		BytesTotal: st.BytesTotal,
	}
}

// FormatProgress returns a single-line progress status.
func FormatProgress(p Progress) string {
	return fmt.Sprintf(
		"Processed=%d Output=%d Skipped=%d Speed=%.0f rows/sec ETA=%s Percent=%.1f%%",
		p.Processed, p.Output, p.Skipped, p.RowsPerSec, p.ETA.Round(time.Second), p.Percent,
	)
}

// FormatSummary returns a human-readable completion summary.
func FormatSummary(output string, st Statistics) string {
	elapsed := st.FinishedAt.Sub(st.StartedAt)
	if st.FinishedAt.IsZero() {
		elapsed = time.Since(st.StartedAt)
	}
	return fmt.Sprintf(
		"Input Rows: %d\nOutput Rows: %d\nSkipped Rows: %d\nElapsed Time: %s\nAverage Speed: %.0f rows/sec\nOutput File: %s",
		st.InputRows, st.OutputRows, st.SkippedRows, elapsed.Round(time.Millisecond), st.RowsPerSec, output,
	)
}
