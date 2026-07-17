package sqlite

import (
	"fmt"
	"time"
)

const sqliteTimeLayout = time.RFC3339Nano

// formatTime encodes a time value for SQLite TEXT storage in UTC.
func formatTime(value time.Time) string {
	return value.UTC().Format(sqliteTimeLayout)
}

// parseTime decodes a SQLite TEXT timestamp into a UTC time.Time.
func parseTime(value string) (time.Time, error) {
	parsed, err := time.Parse(sqliteTimeLayout, value)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, value)
		if err != nil {
			return time.Time{}, fmt.Errorf("sqlite: parse time %q: %w", value, err)
		}
	}
	return parsed.UTC(), nil
}
