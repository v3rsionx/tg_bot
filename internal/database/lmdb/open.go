package lmdb

import "context"

// OpenPath opens an LMDB engine at path using production defaults.
func OpenPath(ctx context.Context, path string) (*DB, error) {
	return OpenDB(ctx, Config{Path: path})
}
