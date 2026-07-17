package importer

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidRecord indicates a source row failed validation.
	ErrInvalidRecord = errors.New("importer: invalid record")
	// ErrMalformedLine indicates a source line could not be parsed.
	ErrMalformedLine = errors.New("importer: malformed line")
	// ErrClosed indicates the importer is no longer usable.
	ErrClosed = errors.New("importer: closed")
)

// errStores returns a missing destination store error.
func errStores(name string) error {
	return fmt.Errorf("importer: Stores.%s is required", name)
}
