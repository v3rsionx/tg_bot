package lmdb

import (
	"errors"

	rawlmdb "github.com/bmatsuo/lmdb-go/lmdb"
)

// validateKey ensures a key is usable by LMDB.
func validateKey(key []byte) error {
	if len(key) == 0 {
		return ErrInvalidKey
	}
	return nil
}

// validateKeyValue ensures both key and value are usable by LMDB.
func validateKeyValue(key, value []byte) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if value == nil {
		return ErrNilValue
	}
	return nil
}

// cloneBytes copies LMDB-managed memory so callers can use it after the transaction ends.
func cloneBytes(src []byte) []byte {
	if src == nil {
		return nil
	}
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

// isNotFound reports whether err represents a missing LMDB key.
func isNotFound(err error) bool {
	return rawlmdb.IsNotFound(err)
}

// isMapFull reports whether err indicates the LMDB map must grow.
// Unwraps fmt.Errorf wrappers so BatchPut/Get paths still trigger auto-growth.
func isMapFull(err error) bool {
	for err != nil {
		if rawlmdb.IsMapFull(err) {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}

// mapNotFound converts LMDB not-found errors into ErrNotFound.
func mapNotFound(err error) error {
	if err == nil {
		return nil
	}
	if isNotFound(err) {
		return ErrNotFound
	}
	return err
}
