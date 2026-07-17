package search

import "errors"

var (
	// ErrNotFound indicates that no exact match exists for the query.
	ErrNotFound = errors.New("search: not found")
	// ErrInvalidQuery indicates that the caller supplied an unusable query.
	ErrInvalidQuery = errors.New("search: invalid query")
	// ErrClosed indicates that the search engine has been closed.
	ErrClosed = errors.New("search: closed")
	// ErrTimeout indicates that the search exceeded its configured timeout.
	ErrTimeout = errors.New("search: timeout")
)
