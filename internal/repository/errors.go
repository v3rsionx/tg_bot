package repository

import "errors"

var (
	// ErrNotFound indicates that the requested entity does not exist.
	ErrNotFound = errors.New("repository: not found")
	// ErrConflict indicates that the requested write conflicts with existing data.
	ErrConflict = errors.New("repository: conflict")
)
