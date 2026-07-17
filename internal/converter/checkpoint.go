package converter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// FileCheckpoint stores resume state for one source file.
type FileCheckpoint struct {
	InputFile    string `json:"input_file"`
	OutputFile   string `json:"output_file"`
	ByteOffset   int64  `json:"byte_offset"`
	LineNumber   uint64 `json:"line_number"`
	InputRows    uint64 `json:"input_rows"`
	OutputRows   uint64 `json:"output_rows"`
	SkippedRows  uint64 `json:"skipped_rows"`
	HeaderDone   bool   `json:"header_done"`
	Encoding     string `json:"encoding"`
	Delimiter    string `json:"delimiter"`
}

// CheckpointStore persists resume metadata as JSON.
type CheckpointStore struct {
	mu   sync.Mutex
	path string
	data map[string]FileCheckpoint
}

// NewCheckpointStore constructs a store for path.
func NewCheckpointStore(path string) *CheckpointStore {
	return &CheckpointStore{
		path: path,
		data: make(map[string]FileCheckpoint),
	}
}

// Load reads checkpoint data from disk when present.
func (s *CheckpointStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return wrap("load checkpoint", err)
	}
	if len(raw) == 0 {
		return nil
	}
	var data map[string]FileCheckpoint
	if err := json.Unmarshal(raw, &data); err != nil {
		return wrap("parse checkpoint", err)
	}
	s.data = data
	return nil
}

// Get returns a checkpoint for file.
func (s *CheckpointStore) Get(file string) (FileCheckpoint, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp, ok := s.data[file]
	return cp, ok
}

// Set updates a checkpoint entry.
func (s *CheckpointStore) Set(cp FileCheckpoint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[cp.InputFile] = cp
}

// Save writes checkpoint data to disk.
func (s *CheckpointStore) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return wrap("create checkpoint dir", err)
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return wrap("marshal checkpoint", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return wrap("write checkpoint", err)
	}
	return wrap("rename checkpoint", os.Rename(tmp, s.path))
}

// Clear removes a file entry after successful conversion.
func (s *CheckpointStore) Clear(file string) error {
	s.mu.Lock()
	delete(s.data, file)
	s.mu.Unlock()
	return s.Save()
}

func checkpointPathFor(dir, input string) string {
	base := filepath.Base(input)
	return filepath.Join(dir, base+".checkpoint.json")
}
