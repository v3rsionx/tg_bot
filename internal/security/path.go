package security

import (
	"path/filepath"
	"strings"
)

// PreventPathTraversal rejects traversal, null bytes, and out-of-root paths.
func (s *Standard) PreventPathTraversal(field, path string) (string, error) {
	if err := s.RejectInvalidUTF8(field, path); err != nil {
		return "", err
	}
	if strings.ContainsRune(path, 0) {
		return "", Error{Field: field, Message: "contains null bytes"}
	}
	if strings.ContainsAny(path, "\r\n") {
		return "", Error{Field: field, Message: "contains newlines"}
	}
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", Error{Field: field, Message: "is required"}
	}
	if len(trimmed) > MaxPathBytes {
		return "", Error{Field: field, Message: "exceeds maximum path length"}
	}
	if hasTraversalSegment(trimmed) {
		return "", Error{Field: field, Message: "path traversal is not allowed"}
	}

	cleaned := filepath.Clean(trimmed)
	if cleaned == "" || cleaned == "." {
		return "", Error{Field: field, Message: "is required"}
	}
	if hasTraversalSegment(cleaned) {
		return "", Error{Field: field, Message: "path traversal is not allowed"}
	}

	if len(s.AllowedRoots) > 0 {
		if err := confineToRoots(field, cleaned, s.AllowedRoots); err != nil {
			return "", err
		}
	}
	return cleaned, nil
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

func confineToRoots(field, cleaned string, roots []string) error {
	absPath, err := filepath.Abs(cleaned)
	if err != nil {
		return Error{Field: field, Message: "cannot resolve absolute path"}
	}
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		absRoot, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(absRoot, absPath)
		if err != nil || filepath.IsAbs(rel) {
			continue
		}
		if rel == "." || !isParentRel(rel) {
			return nil
		}
	}
	return Error{Field: field, Message: "path is outside allowed roots"}
}

func isParentRel(rel string) bool {
	if rel == ".." {
		return true
	}
	sep := string(filepath.Separator)
	return strings.HasPrefix(rel, ".."+sep)
}
