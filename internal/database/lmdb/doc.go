// Package lmdb implements a production-oriented LMDB key-value storage engine.
//
// The package provides thread-safe access, read-only and read-write
// transactions, automatic map growth, cursor iteration, and batch helpers.
// It does not implement search, import, or domain business logic.
package lmdb
