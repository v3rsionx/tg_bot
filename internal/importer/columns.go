package importer

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const unsetColumn = -1

// ColumnMapping maps logical fields to zero-based CSV column indexes.
// Unset optional columns use -1.
type ColumnMapping struct {
	ID       int
	Name     int
	Phone    int
	Username int
	Extras   int
	// Source describes how the mapping was resolved.
	Source string
}

// HasName reports whether a name column is mapped.
func (m ColumnMapping) HasName() bool { return m.Name >= 0 }

// HasExtras reports whether an extras column is mapped.
func (m ColumnMapping) HasExtras() bool { return m.Extras >= 0 }

func mappingFromConfig(cfg Config) ColumnMapping {
	name := cfg.NameColumn
	if name < 0 {
		name = unsetColumn
	}
	extras := cfg.ExtrasColumn
	if extras < 0 {
		extras = unsetColumn
	}
	return ColumnMapping{
		ID:       cfg.IDColumn,
		Name:     name,
		Phone:    cfg.PhoneColumn,
		Username: cfg.UsernameColumn,
		Extras:   extras,
		Source:   "config",
	}
}

// resolveHeaderMapping builds a mapping from a header row when recognized.
// ok is false when the row does not look like a supported header.
func resolveHeaderMapping(headers []string) (ColumnMapping, bool) {
	m := ColumnMapping{
		ID:       unsetColumn,
		Name:     unsetColumn,
		Phone:    unsetColumn,
		Username: unsetColumn,
		Extras:   unsetColumn,
	}
	known := 0
	for i, h := range headers {
		switch normalizeHeaderName(h) {
		case "id":
			if m.ID < 0 {
				m.ID = i
				known++
			}
		case "name":
			if m.Name < 0 {
				m.Name = i
				known++
			}
		case "phone":
			if m.Phone < 0 {
				m.Phone = i
				known++
			}
		case "username":
			if m.Username < 0 {
				m.Username = i
				known++
			}
		case "extras":
			if m.Extras < 0 {
				m.Extras = i
				known++
			}
		}
	}
	if m.ID < 0 || (m.Phone < 0 && m.Username < 0) {
		return ColumnMapping{}, false
	}
	switch {
	case m.HasName() && m.HasExtras() && m.Phone >= 0 && m.Username >= 0:
		m.Source = "header:standard"
	case !m.HasName() && !m.HasExtras() && m.Phone >= 0 && m.Username >= 0:
		m.Source = "header:legacy"
	default:
		m.Source = "header:partial"
	}
	if known == 0 {
		return ColumnMapping{}, false
	}
	return m, true
}

func normalizeHeaderName(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "-", "")
	return s
}

// peekHeaderMapping reads the first non-empty line of path and attempts header mapping.
func peekHeaderMapping(path string, delimiter rune, bufferSize int) (ColumnMapping, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return ColumnMapping{}, false, fmt.Errorf("importer: open header %q: %w", path, err)
	}
	defer f.Close()

	reader := bufio.NewReaderSize(f, bufferSize)
	for {
		line, err := reader.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		if strings.TrimSpace(line) != "" {
			fields, perr := parseFields(line, delimiter)
			if perr != nil {
				return ColumnMapping{}, false, nil
			}
			m, ok := resolveHeaderMapping(fields)
			return m, ok, nil
		}
		if err != nil {
			return ColumnMapping{}, false, nil
		}
	}
}
