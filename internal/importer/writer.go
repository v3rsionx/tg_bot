package importer

import (
	"context"
	"fmt"

	"github.com/v3rsionx/tg_bot/internal/database/lmdb"
)

// indexWriter performs batched exact-lookup writes across destination stores.
type indexWriter struct {
	stores           Stores
	batchSize        int
	skipDuplicateIDs bool
	updateExisting   bool
	stats            *statsAccumulator
	checkpoint       *CheckpointStore
	log              Logger

	idBatch       []lmdb.KeyValue
	phoneBatch    []lmdb.KeyValue
	usernameBatch []lmdb.KeyValue
	lastFile      string
	lastOffset    int64
	lastLine      uint64
}

// newIndexWriter constructs a batching index writer.
func newIndexWriter(
	stores Stores,
	cfg Config,
	stats *statsAccumulator,
	checkpoint *CheckpointStore,
	log Logger,
) *indexWriter {
	return &indexWriter{
		stores:           stores,
		batchSize:        cfg.BatchSize,
		skipDuplicateIDs: cfg.SkipDuplicateIDs,
		updateExisting:   cfg.UpdateExisting,
		stats:            stats,
		checkpoint:       checkpoint,
		log:              log,
		idBatch:          make([]lmdb.KeyValue, 0, cfg.BatchSize),
		phoneBatch:       make([]lmdb.KeyValue, 0, cfg.BatchSize),
		usernameBatch:    make([]lmdb.KeyValue, 0, cfg.BatchSize),
	}
}

// Handle decides insert/update/skip and buffers index writes.
func (w *indexWriter) Handle(ctx context.Context, record Record) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	exists, err := w.stores.ID.Exists(ctx, []byte(record.ID))
	if err != nil {
		return fmt.Errorf("importer: exists id %s: %w", record.ID, err)
	}

	update := false
	switch {
	case exists && w.skipDuplicateIDs:
		w.stats.duplicates.Add(1)
		w.trackCheckpoint(record)
		return nil
	case exists && w.updateExisting:
		update = true
	case exists:
		w.stats.duplicates.Add(1)
		w.trackCheckpoint(record)
		return nil
	}

	if update {
		existing, getErr := w.stores.ID.Get(ctx, []byte(record.ID))
		if getErr != nil {
			return fmt.Errorf("importer: get existing id %s: %w", record.ID, getErr)
		}
		oldPhone, oldUsername, oldName, oldExtras := decodeIDPayload(existing)
		record.Extras = mergeExtrasJSON(oldExtras, record.Extras)
		// Keep prior contact fields when the new dump left them empty.
		if record.Phone == "" {
			record.Phone = oldPhone
		}
		if record.Username == "" {
			record.Username = oldUsername
		}
		if record.Name == "" {
			record.Name = oldName
		}
	}

	if record.Extras != "" && record.Extras != "{}" {
		w.stats.extrasRetained.Add(1)
	}

	payload := encodeIDPayload(record.Phone, record.Username, record.Name, record.Extras)
	w.idBatch = append(w.idBatch, lmdb.KeyValue{
		Key:   []byte(record.ID),
		Value: payload,
	})
	if record.Phone != "" {
		w.phoneBatch = append(w.phoneBatch, lmdb.KeyValue{
			Key:   []byte(record.Phone),
			Value: []byte(record.ID),
		})
		w.stats.phoneWrites.Add(1)
	}
	if record.Username != "" {
		w.usernameBatch = append(w.usernameBatch, lmdb.KeyValue{
			Key:   []byte(record.Username),
			Value: []byte(record.ID),
		})
		w.stats.usernameWrites.Add(1)
	}

	if update {
		w.stats.updates.Add(1)
	} else {
		w.stats.inserts.Add(1)
	}
	w.trackCheckpoint(record)

	if len(w.idBatch) >= w.batchSize {
		return w.Flush(ctx)
	}
	return nil
}

// Flush writes buffered batches and persists the checkpoint.
func (w *indexWriter) Flush(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(w.idBatch) == 0 && len(w.phoneBatch) == 0 && len(w.usernameBatch) == 0 {
		return w.saveCheckpoint()
	}

	if len(w.idBatch) > 0 {
		if err := w.stores.ID.BatchPut(ctx, w.idBatch); err != nil {
			return fmt.Errorf("importer: batch put id: %w", err)
		}
		w.idBatch = w.idBatch[:0]
	}
	if len(w.phoneBatch) > 0 {
		if err := w.stores.Phone.BatchPut(ctx, w.phoneBatch); err != nil {
			return fmt.Errorf("importer: batch put phone: %w", err)
		}
		w.phoneBatch = w.phoneBatch[:0]
	}
	if len(w.usernameBatch) > 0 {
		if err := w.stores.Username.BatchPut(ctx, w.usernameBatch); err != nil {
			return fmt.Errorf("importer: batch put username: %w", err)
		}
		w.usernameBatch = w.usernameBatch[:0]
	}

	w.stats.batchesWritten.Add(1)
	return w.saveCheckpoint()
}

// trackCheckpoint remembers the latest successfully accepted record position.
func (w *indexWriter) trackCheckpoint(record Record) {
	w.lastFile = record.File
	w.lastOffset = record.Offset
	w.lastLine = record.Line
}

// saveCheckpoint persists resume metadata when configured.
func (w *indexWriter) saveCheckpoint() error {
	if w.checkpoint == nil || w.lastFile == "" {
		return nil
	}
	w.checkpoint.Set(w.lastFile, w.lastOffset, w.lastLine)
	if err := w.checkpoint.Save(); err != nil {
		return err
	}
	return nil
}

// encodeIDPayload stores companions for an ID key as:
//
//	phone\0username\0name\0extras
func encodeIDPayload(phone, username, name, extras string) []byte {
	buf := make([]byte, 0, len(phone)+1+len(username)+1+len(name)+1+len(extras))
	buf = append(buf, phone...)
	buf = append(buf, 0)
	buf = append(buf, username...)
	buf = append(buf, 0)
	buf = append(buf, name...)
	buf = append(buf, 0)
	buf = append(buf, extras...)
	return buf
}
