// Package converter streams arbitrary CSV/TXT dumps into a standard CSV
// layout suitable for the existing importer:
//
//	id,name,phone,username,extras
//
// It detects encodings and delimiters, maps known header aliases, and packs
// unknown columns into a JSON extras field. Processing is streaming-only and
// designed for multi-tens-of-millions of rows.
//
// Callers inject Config, Logger, and ProgressFunc. There is no package-level
// mutable state.
package converter
