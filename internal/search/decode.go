package search

import (
	"bytes"
	"fmt"
)

// decodeIDPayload parses the importer ID payload format:
//
//	phone\0username\0name\0extras
//
// Legacy payloads phone\0username (2 fields) remain valid; name and extras
// are empty in that case.
func decodeIDPayload(id string, payload []byte) (Record, error) {
	if payload == nil {
		return Record{}, fmt.Errorf("%w: empty payload", ErrNotFound)
	}
	parts := bytes.SplitN(payload, []byte{0}, 4)
	record := Record{ID: id}
	if len(parts) > 0 {
		record.Phone = string(parts[0])
	}
	if len(parts) > 1 {
		record.Username = string(parts[1])
	}
	if len(parts) > 2 {
		record.Name = string(parts[2])
	}
	if len(parts) > 3 {
		record.Extras = string(parts[3])
	}
	return record, nil
}

// errStore returns a missing store configuration error.
func errStore(name string) error {
	return fmt.Errorf("search: Stores.%s is required", name)
}
