package validator

import (
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// SQLitePath validates a SQLite database file path.
func (v *Standard) SQLitePath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return Error{Field: "SQLITE_PATH", Message: "is required"}
	}
	if err := validatePathValue("SQLITE_PATH", path); err != nil {
		return err
	}
	if filepath.Base(filepath.Clean(path)) == "." {
		return Error{Field: "SQLITE_PATH", Message: "must identify a database file"}
	}
	return nil
}

// LMDBPath validates one LMDB environment directory path.
func (v *Standard) LMDBPath(field, path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return Error{Field: field, Message: "is required"}
	}
	return validatePathValue(field, path)
}

// DistinctPaths validates that paths are non-empty and unique.
func (v *Standard) DistinctPaths(paths map[string]string) error {
	seen := make(map[string]string, len(paths))
	for field, path := range paths {
		cleaned := filepath.Clean(strings.TrimSpace(path))
		if cleaned == "" || cleaned == "." {
			return Error{Field: field, Message: "is required"}
		}
		key := filepath.ToSlash(strings.ToLower(cleaned))
		if other, exists := seen[key]; exists {
			return Error{Field: field, Message: "must be distinct from " + other}
		}
		seen[key] = field
	}
	return nil
}

func validatePathValue(field, path string) error {
	if !utf8.ValidString(path) {
		return Error{Field: field, Message: "contains invalid UTF-8"}
	}
	if strings.ContainsRune(path, 0) {
		return Error{Field: field, Message: "contains null bytes"}
	}
	if strings.ContainsAny(path, "\r\n") {
		return Error{Field: field, Message: "must not contain newlines"}
	}
	if len(path) > 4096 {
		return Error{Field: field, Message: "exceeds maximum path length"}
	}
	cleaned := filepath.Clean(path)
	if cleaned == "" || cleaned == "." {
		return Error{Field: field, Message: "is required"}
	}
	if hasTraversalSegment(path) || hasTraversalSegment(cleaned) {
		return Error{Field: field, Message: "must not contain path traversal"}
	}
	return nil
}

func hasTraversalSegment(path string) bool {
	normalized := filepath.ToSlash(path)
	normalized = strings.ReplaceAll(normalized, `\`, "/")
	for _, part := range strings.Split(normalized, "/") {
		if part == ".." {
			return true
		}
	}
	return false
}
