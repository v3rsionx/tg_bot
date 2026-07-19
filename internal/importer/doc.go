// Package importer provides a streaming, resumable import pipeline for
// exact-lookup LMDB indexes (id, phone, username).
//
// Supported source layouts:
//   - standard header: id,name,phone,username,extras (converter output)
//   - legacy header:   id,phone,username
//   - positional:      configured IDColumn/PhoneColumn/UsernameColumn
//
// ID is mandatory. Phone, username, name, and extras are optional; present
// phone/username values are still validated. Name and extras are retained in
// memory; LMDB payload encoding remains phone\0username until persistence
// support is added.
//
// The package is designed for multi-file CSV/TXT sources larger than 100GB
// using bounded memory, worker pools, batch writes, and graceful shutdown.
package importer
