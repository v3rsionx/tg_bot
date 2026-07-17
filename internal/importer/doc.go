// Package importer provides a streaming, resumable import pipeline for
// exact-lookup LMDB indexes (id, phone, username).
//
// The package is designed for multi-file CSV/TXT sources larger than 100GB
// using bounded memory, worker pools, batch writes, and graceful shutdown.
package importer
