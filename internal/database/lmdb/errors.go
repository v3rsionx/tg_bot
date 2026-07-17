package lmdb

import "errors"

var (
	// ErrClosed indicates that the environment has been closed.
	ErrClosed = errors.New("lmdb: environment is closed")
	// ErrNotFound indicates that the requested key does not exist.
	ErrNotFound = errors.New("lmdb: key not found")
	// ErrTxnClosed indicates that the transaction is no longer usable.
	ErrTxnClosed = errors.New("lmdb: transaction is closed")
	// ErrReadOnly indicates that a write was attempted on a read-only transaction.
	ErrReadOnly = errors.New("lmdb: transaction is read-only")
	// ErrMapSizeLimit indicates that automatic map growth reached the configured ceiling.
	ErrMapSizeLimit = errors.New("lmdb: map size limit reached")
	// ErrInvalidKey indicates that a key is empty or otherwise invalid.
	ErrInvalidKey = errors.New("lmdb: invalid key")
	// ErrNilValue indicates that a nil value was provided where bytes are required.
	ErrNilValue = errors.New("lmdb: nil value")
)
