package converter

import "strings"

// buildMapping maps source headers to the standard schema.
// The first matching alias wins for each known role; remaining columns become extras.
func buildMapping(headers []string) ColumnMapping {
	m := ColumnMapping{
		IDIndex:       -1,
		NameIndex:     -1,
		LastNameIndex: -1,
		PhoneIndex:    -1,
		UsernameIndex: -1,
		Headers:       append([]string(nil), headers...),
	}
	for i, h := range headers {
		role, ok := classifyHeader(h)
		if !ok {
			appendExtra(&m, i, h)
			continue
		}
		switch role {
		case RoleID:
			if m.IDIndex < 0 {
				m.IDIndex = i
			} else {
				appendExtra(&m, i, h)
			}
		case RoleName:
			if m.NameIndex < 0 {
				m.NameIndex = i
			} else {
				appendExtra(&m, i, h)
			}
		case RoleLastName:
			if m.LastNameIndex < 0 {
				m.LastNameIndex = i
			} else {
				appendExtra(&m, i, h)
			}
		case RolePhone:
			if m.PhoneIndex < 0 {
				m.PhoneIndex = i
			} else {
				appendExtra(&m, i, h)
			}
		case RoleUsername:
			if m.UsernameIndex < 0 {
				m.UsernameIndex = i
			} else {
				appendExtra(&m, i, h)
			}
		}
	}
	return m
}

func appendExtra(m *ColumnMapping, i int, h string) {
	m.ExtrasIndexes = append(m.ExtrasIndexes, i)
	m.ExtrasNames = append(m.ExtrasNames, h)
}

// combineName joins first and last name fields.
func combineName(first, last string) string {
	first = strings.TrimSpace(first)
	last = strings.TrimSpace(last)
	switch {
	case first == "":
		return last
	case last == "":
		return first
	default:
		return first + " " + last
	}
}

func fieldAt(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}
