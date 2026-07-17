package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

func formatText(entry Entry) []byte {
	var b strings.Builder
	b.WriteString(entry.Timestamp.Format(time.RFC3339Nano))
	b.WriteByte(' ')
	b.WriteString(strings.ToUpper(entry.Level))
	b.WriteByte(' ')
	b.WriteString(entry.Message)
	if entry.CorrelationID != "" {
		b.WriteString(" correlation_id=")
		b.WriteString(entry.CorrelationID)
	}
	if len(entry.Fields) > 0 {
		keys := make([]string, 0, len(entry.Fields))
		for k := range entry.Fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteByte(' ')
			b.WriteString(k)
			b.WriteByte('=')
			b.WriteString(fmt.Sprint(entry.Fields[k]))
		}
	}
	b.WriteByte('\n')
	return []byte(b.String())
}

func formatJSON(entry Entry) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(entry)
	return buf.Bytes()
}
