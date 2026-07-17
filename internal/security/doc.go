// Package security provides input sanitization and normalization helpers.
//
// It contains no business logic and no Telegram transport logic. Callers inject
// Sanitizer implementations where untrusted input must be hardened.
package security
