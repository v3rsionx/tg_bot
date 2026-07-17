package importer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Checkpoint persists per-file resume offsets.
type Checkpoint struct {
	Files map[string]FileCheckpoint `json:"files"`
}

// FileCheckpoint stores resume metadata for one source file.
type FileCheckpoint struct {
	Offset int64  `json:"offset"`
	Line   uint64 `json:"line"`
}

// CheckpointStore loads and saves importer checkpoints.
type CheckpointStore struct {
	path string
	mu   sync.Mutex
	data Checkpoint
}

// NewCheckpointStore constructs a checkpoint store for path.
func NewCheckpointStore(path string) *CheckpointStore {
	return &CheckpointStore{
		path: path,
		data: Checkpoint{Files: make(map[string]FileCheckpoint)},
	}
}

// Load reads an existing checkpoint file when present.
func (s *CheckpointStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.path == "" {
		return nil
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("importer: read checkpoint: %w", err)
	}
	var data Checkpoint
	if err := json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("importer: parse checkpoint: %w", err)
	}
	if data.Files == nil {
		data.Files = make(map[string]FileCheckpoint)
	}
	s.data = data
	return nil
}

// Get returns the checkpoint for file.
func (s *CheckpointStore) Get(file string) FileCheckpoint {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.Files[file]
}

// Set updates the in-memory checkpoint for file.
func (s *CheckpointStore) Set(file string, offset int64, line uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Files == nil {
		s.data.Files = make(map[string]FileCheckpoint)
	}
	s.data.Files[file] = FileCheckpoint{Offset: offset, Line: line}
}

// Save writes the checkpoint atomically to disk.
func (s *CheckpointStore) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("importer: create checkpoint directory: %w", err)
	}

	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("importer: encode checkpoint: %w", err)
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("importer: write checkpoint temp: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("importer: rename checkpoint: %w", err)
	}
	return nil
}
