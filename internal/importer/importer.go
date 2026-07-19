package importer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

// Importer streams source files into exact-lookup LMDB indexes.
type Importer struct {
	cfg        Config
	stores     Stores
	log        Logger
	progressFn ProgressFunc
	validator  *Validator
	checkpoint *CheckpointStore
}

// New constructs an Importer with dependency-injected stores and logger.
func New(cfg Config, stores Stores, log Logger, progressFn ProgressFunc) (*Importer, error) {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if err := stores.Validate(); err != nil {
		return nil, err
	}
	if log == nil {
		log = NopLogger{}
	}

	var checkpoint *CheckpointStore
	if cfg.CheckpointPath != "" {
		checkpoint = NewCheckpointStore(cfg.CheckpointPath)
		if cfg.Resume {
			if err := checkpoint.Load(); err != nil {
				return nil, err
			}
		}
	}

	return &Importer{
		cfg:        cfg,
		stores:     stores,
		log:        log,
		progressFn: progressFn,
		validator:  NewValidator(cfg),
		checkpoint: checkpoint,
	}, nil
}

// Run imports all configured sources until completion, cancellation, or error.
func (im *Importer) Run(ctx context.Context) (Statistics, error) {
	stats := newStatsAccumulator(len(im.cfg.Sources))
	defer stats.finish()

	progressCtx, cancelProgress := context.WithCancel(ctx)
	defer cancelProgress()

	var progressWG sync.WaitGroup
	if im.progressFn != nil {
		progressWG.Add(1)
		go func() {
			defer progressWG.Done()
			im.reportProgress(progressCtx, stats)
		}()
	}

	var runErr error
	for _, source := range im.cfg.Sources {
		if err := ctx.Err(); err != nil {
			runErr = err
			break
		}
		if err := im.importFile(ctx, source, stats); err != nil {
			runErr = err
			break
		}
		stats.markFileCompleted()
	}

	cancelProgress()
	progressWG.Wait()

	final := stats.snapshot(time.Now().UTC())
	if im.progressFn != nil {
		im.progressFn(Progress{Statistics: final})
	}
	return final, runErr
}

// importFile streams one source file through the worker pool and batch writer.
func (im *Importer) importFile(ctx context.Context, path string, stats *statsAccumulator) error {
	var resumeOffset int64
	var resumeLine uint64
	if im.cfg.Resume && im.checkpoint != nil {
		cp := im.checkpoint.Get(path)
		resumeOffset = cp.Offset
		resumeLine = cp.Line
	}

	reader, total, err := openLineReader(path, resumeOffset, im.cfg.ReadBufferBytes)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()

	stats.addBytesTotal(total)
	if resumeOffset > 0 {
		stats.bytesRead.Add(resumeOffset)
	}

	mapping, skipDetectedHeader, err := im.resolveMapping(path)
	if err != nil {
		return err
	}
	im.validator.SetMapping(mapping)
	im.log.Infof("importing %s (resume offset=%d line=%d size=%d mapping=%s id=%d name=%d phone=%d username=%d extras=%d)",
		path, resumeOffset, resumeLine, total, mapping.Source,
		mapping.ID, mapping.Name, mapping.Phone, mapping.Username, mapping.Extras)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan rawJob, im.cfg.QueueSize)
	writes := make(chan Record, im.cfg.QueueSize)
	errCh := make(chan error, 1)

	sendErr := func(err error) {
		if err == nil || errors.Is(err, context.Canceled) {
			return
		}
		select {
		case errCh <- err:
		default:
		}
		cancel()
	}

	var workerWG sync.WaitGroup
	for i := 0; i < im.cfg.Workers; i++ {
		workerWG.Add(1)
		go func() {
			defer workerWG.Done()
			sendErr(im.worker(ctx, jobs, writes, stats))
		}()
	}

	var writerWG sync.WaitGroup
	writerWG.Add(1)
	go func() {
		defer writerWG.Done()
		sendErr(im.writeLoop(ctx, writes, stats))
	}()

	readErr := im.readLoop(ctx, path, reader, resumeOffset, resumeLine, skipDetectedHeader, jobs, stats)
	close(jobs)
	workerWG.Wait()
	close(writes)
	writerWG.Wait()

	if retained := stats.extrasRetained.Load(); retained > 0 {
		im.log.Infof("%s: persisted extras for %d rows", path, retained)
	}

	select {
	case err := <-errCh:
		return err
	default:
	}
	if readErr != nil && !errors.Is(readErr, io.EOF) && !errors.Is(readErr, context.Canceled) {
		return readErr
	}
	if errors.Is(readErr, context.Canceled) {
		return readErr
	}
	return nil
}

// resolveMapping chooses header-based or config-based column mapping for path.
func (im *Importer) resolveMapping(path string) (ColumnMapping, bool, error) {
	fallback := mappingFromConfig(im.cfg)
	if !im.cfg.AutoMapHeaders && !im.cfg.HasHeader {
		return fallback, false, nil
	}
	if !im.cfg.AutoMapHeaders {
		// Explicit HasHeader with fixed indexes: skip header, keep config mapping.
		return fallback, im.cfg.HasHeader, nil
	}
	detected, ok, err := peekHeaderMapping(path, im.cfg.Delimiter, im.cfg.ReadBufferBytes)
	if err != nil {
		return ColumnMapping{}, false, err
	}
	if ok {
		return detected, true, nil
	}
	return fallback, im.cfg.HasHeader, nil
}

// readLoop streams source lines into the worker queue.
func (im *Importer) readLoop(
	ctx context.Context,
	path string,
	reader *lineReader,
	resumeOffset int64,
	resumeLine uint64,
	skipHeader bool,
	jobs chan<- rawJob,
	stats *statsAccumulator,
) error {
	lineNo := resumeLine
	skipHeader = skipHeader && resumeLine == 0
	prevOffset := resumeOffset

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		text, nextOffset, err := reader.ReadLine(im.cfg.MaxLineBytes)
		if nextOffset > prevOffset {
			stats.bytesRead.Add(nextOffset - prevOffset)
			prevOffset = nextOffset
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			if errors.Is(err, ErrMalformedLine) {
				stats.recordsInvalid.Add(1)
				im.log.Warnf("%s: %v", path, err)
				continue
			}
			return err
		}

		lineNo++
		stats.linesRead.Add(1)
		if text == "" {
			continue
		}
		if skipHeader {
			skipHeader = false
			im.log.Debugf("skipped header in %s", path)
			continue
		}

		job := rawJob{
			File:   path,
			Line:   lineNo,
			Offset: nextOffset,
			Text:   text,
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case jobs <- job:
		}
	}
}

// worker parses and validates raw lines.
func (im *Importer) worker(
	ctx context.Context,
	jobs <-chan rawJob,
	writes chan<- Record,
	stats *statsAccumulator,
) error {
	for job := range jobs {
		fields, err := parseFields(job.Text, im.cfg.Delimiter)
		if err != nil {
			stats.recordsInvalid.Add(1)
			im.log.Warnf("%s:%d malformed: %v", job.File, job.Line, err)
			continue
		}
		stats.recordsParsed.Add(1)

		record, err := im.validator.ValidateFields(fields, Record{
			File:   job.File,
			Line:   job.Line,
			Offset: job.Offset,
		})
		if err != nil {
			stats.recordsInvalid.Add(1)
			im.log.Debugf("%s:%d invalid: %v", job.File, job.Line, err)
			continue
		}

		select {
		case writes <- record:
		case <-ctx.Done():
			// Keep draining jobs after cancellation to avoid deadlocking the reader.
			continue
		}
	}
	return nil
}

// writeLoop consumes validated records and performs batched LMDB writes.
// It exits only after the writes channel is closed so in-flight records are drained.
func (im *Importer) writeLoop(ctx context.Context, writes <-chan Record, stats *statsAccumulator) error {
	writer := newIndexWriter(im.stores, im.cfg, stats, im.checkpoint, im.log)
	for record := range writes {
		handleCtx := ctx
		if err := ctx.Err(); err != nil {
			handleCtx = context.Background()
		}
		if err := writer.Handle(handleCtx, record); err != nil {
			return err
		}
	}
	return writer.Flush(context.Background())
}

// reportProgress emits periodic progress snapshots until ctx is cancelled.
func (im *Importer) reportProgress(ctx context.Context, stats *statsAccumulator) {
	ticker := time.NewTicker(im.cfg.ProgressInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			snapshot := stats.snapshot(time.Now().UTC())
			im.progressFn(Progress{Statistics: snapshot})
			im.log.Infof(
				"progress inserts=%d updates=%d dupes=%d invalid=%d rate=%.0f rec/s eta=%s",
				snapshot.Inserts,
				snapshot.Updates,
				snapshot.Duplicates,
				snapshot.RecordsInvalid,
				snapshot.RecordsPerSecond,
				snapshot.ETA.Round(time.Second),
			)
		}
	}
}

// FormatStatistics returns a concise human-readable statistics summary.
func FormatStatistics(stats Statistics) string {
	return fmt.Sprintf(
		"files=%d/%d bytes=%d/%d lines=%d inserts=%d updates=%d duplicates=%d invalid=%d batches=%d extras_retained=%d rate=%.0f/s",
		stats.FilesCompleted,
		stats.FilesTotal,
		stats.BytesRead,
		stats.BytesTotal,
		stats.LinesRead,
		stats.Inserts,
		stats.Updates,
		stats.Duplicates,
		stats.RecordsInvalid,
		stats.BatchesWritten,
		stats.ExtrasRetained,
		stats.RecordsPerSecond,
	)
}
