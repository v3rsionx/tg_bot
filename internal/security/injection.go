package security

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	envKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	sqlDangerous  = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bunion\s+all\s+select\b`),
		regexp.MustCompile(`(?i)\bunion\s+select\b`),
		regexp.MustCompile(`(?i)\bdrop\s+table\b`),
		regexp.MustCompile(`(?i)\binsert\s+into\b`),
		regexp.MustCompile(`(?i)\bdelete\s+from\b`),
		regexp.MustCompile(`(?i)\bupdate\s+\w+\s+set\b`),
		regexp.MustCompile(`(?i)\bor\s+1\s*=\s*1\b`),
		regexp.MustCompile(`(?i)\band\s+1\s*=\s*1\b`),
		regexp.MustCompile(`(?i)\bsleep\s*\(`),
		regexp.MustCompile(`(?i)\bbenchmark\s*\(`),
		regexp.MustCompile(`(?i)\bxp_cmdshell\b`),
		regexp.MustCompile(`(?i)\binformation_schema\b`),
		regexp.MustCompile(`(?i);\s*(drop|delete|insert|update|select)\b`),
		regexp.MustCompile(`--|/\*|\*/`),
		regexp.MustCompile(`(?i)'\s*(or|and)\s+`),
	}
)

// PreventConfigInjection rejects config/env injection patterns.
func (s *Standard) PreventConfigInjection(key, value string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return Error{Field: "config", Message: "key is required"}
	}
	if !envKeyPattern.MatchString(key) {
		return Error{Field: key, Message: "key contains injection characters"}
	}
	if strings.ContainsRune(value, 0) {
		return Error{Field: key, Message: "value contains null bytes"}
	}
	if strings.ContainsAny(value, "\r\n") {
		return Error{Field: key, Message: "value must not contain newlines"}
	}
	if !utf8.ValidString(value) {
		return Error{Field: key, Message: "value contains invalid UTF-8"}
	}
	if len(value) > 1<<20 {
		return Error{Field: key, Message: "value exceeds 1 MiB"}
	}
	lower := strings.ToLower(value)
	if strings.Contains(lower, "${") || strings.Contains(lower, "$(") || strings.Contains(lower, "`") {
		return Error{Field: key, Message: "value contains shell/config expansion syntax"}
	}
	return nil
}

// PreventSQLInjection rejects common SQL injection payloads in free text.
func (s *Standard) PreventSQLInjection(field, value string) error {
	if err := s.RejectInvalidUTF8(field, value); err != nil {
		return err
	}
	if strings.ContainsRune(value, 0) {
		return Error{Field: field, Message: "contains null bytes"}
	}
	for _, re := range sqlDangerous {
		if re.MatchString(value) {
			return Error{Field: field, Message: "contains disallowed SQL patterns"}
		}
	}
	return nil
}
