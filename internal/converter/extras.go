package converter

import (
	"bytes"
	"encoding/json"
	"strings"
)

// buildExtrasJSON packs unknown columns into a compact JSON object.
// Empty values are omitted. Returns "{}" when there is nothing to store.
func buildExtrasJSON(row []string, mapping ColumnMapping) string {
	if len(mapping.ExtrasIndexes) == 0 {
		return "{}"
	}
	m := make(map[string]string, len(mapping.ExtrasIndexes))
	for i, idx := range mapping.ExtrasIndexes {
		name := mapping.ExtrasNames[i]
		if name == "" {
			continue
		}
		val := fieldAt(row, idx)
		if val == "" {
			continue
		}
		m[name] = val
	}
	if len(m) == 0 {
		return "{}"
	}
	data, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// compactJSON ensures stable empty object form.
func compactJSON(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "null" {
		return "{}"
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(s)); err != nil {
		return s
	}
	return buf.String()
}
