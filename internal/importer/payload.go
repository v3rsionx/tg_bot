package importer

import (
	"bytes"
	"encoding/json"
	"strings"
)

// decodeIDPayload parses phone\0username\0name\0extras (legacy 2-field OK).
func decodeIDPayload(payload []byte) (phone, username, name, extras string) {
	if len(payload) == 0 {
		return "", "", "", ""
	}
	parts := bytes.SplitN(payload, []byte{0}, 4)
	if len(parts) > 0 {
		phone = string(parts[0])
	}
	if len(parts) > 1 {
		username = string(parts[1])
	}
	if len(parts) > 2 {
		name = string(parts[2])
	}
	if len(parts) > 3 {
		extras = string(parts[3])
	}
	return phone, username, name, extras
}

// mergeExtrasJSON merges old and new extras objects.
// Existing keys are kept; new keys are added; same key → new value wins.
// Non-object JSON falls back to keeping both under _previous / top-level replace.
func mergeExtrasJSON(oldExtras, newExtras string) string {
	oldExtras = strings.TrimSpace(oldExtras)
	newExtras = strings.TrimSpace(newExtras)
	if oldExtras == "" || oldExtras == "{}" {
		if newExtras == "" {
			return "{}"
		}
		return newExtras
	}
	if newExtras == "" || newExtras == "{}" {
		return oldExtras
	}

	oldMap, oldOK := parseExtrasObject(oldExtras)
	newMap, newOK := parseExtrasObject(newExtras)
	if !oldOK && !newOK {
		return newExtras
	}
	if !oldOK {
		return newExtras
	}
	if !newOK {
		return oldExtras
	}

	merged := make(map[string]any, len(oldMap)+len(newMap))
	for k, v := range oldMap {
		merged[k] = v
	}
	for k, v := range newMap {
		if existing, ok := merged[k]; ok {
			merged[k] = mergeExtrasValue(existing, v)
			continue
		}
		merged[k] = v
	}
	b, err := json.Marshal(merged)
	if err != nil {
		return newExtras
	}
	return string(b)
}

func parseExtrasObject(raw string) (map[string]any, bool) {
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil || m == nil {
		return nil, false
	}
	return m, true
}

func mergeExtrasValue(oldVal, newVal any) any {
	oldObj, oldOK := oldVal.(map[string]any)
	newObj, newOK := newVal.(map[string]any)
	if oldOK && newOK {
		out := make(map[string]any, len(oldObj)+len(newObj))
		for k, v := range oldObj {
			out[k] = v
		}
		for k, v := range newObj {
			if existing, ok := out[k]; ok {
				out[k] = mergeExtrasValue(existing, v)
				continue
			}
			out[k] = v
		}
		return out
	}
	// Scalar / array conflict: keep both when different.
	if extrasValuesEqual(oldVal, newVal) {
		return newVal
	}
	return []any{oldVal, newVal}
}

func extrasValuesEqual(a, b any) bool {
	ab, err1 := json.Marshal(a)
	bb, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return string(ab) == string(bb)
}
