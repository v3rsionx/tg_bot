package search

import (
	"bytes"
	"fmt"
)

// decodeIDPayload parses the importer ID payload format: phone\0username.
func decodeIDPayload(id string, payload []byte) (Record, error) {
	if payload == nil {
		return Record{}, fmt.Errorf("%w: empty payload", ErrNotFound)
	}
	parts := bytes.SplitN(payload, []byte{0}, 2)
	record := Record{ID: id}
	if len(parts) > 0 {
		record.Phone = string(parts[0])
	}
	if len(parts) > 1 {
		record.Username = string(parts[1])
	}
	if record.Phone == "" && record.Username == "" {
		return Record{}, fmt.Errorf("%w: empty record fields", ErrNotFound)
	}
	return record, nil
}

// errStore returns a missing store configuration error.
func errStore(name string) error {
	return fmt.Errorf("search: Stores.%s is required", name)
}
