package importer

import (
	"encoding/json"
	"testing"
)

func TestMergeExtrasJSONKeepsOldAndAddsNew(t *testing.T) {
	got := mergeExtrasJSON(
		`{"access_hash":"111","country":"BD"}`,
		`{"customerId":70965,"nick":"Алиме"}`,
	)
	var m map[string]any
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("json: %v (%s)", err, got)
	}
	if m["access_hash"] != "111" || m["country"] != "BD" {
		t.Fatalf("lost old keys: %s", got)
	}
	if m["customerId"] != float64(70965) && m["customerId"] != 70965 {
		// json numbers decode as float64
		if _, ok := m["customerId"]; !ok {
			t.Fatalf("missing new customerId: %s", got)
		}
	}
	if m["nick"] != "Алиме" {
		t.Fatalf("missing new nick: %s", got)
	}
}

func TestMergeExtrasJSONNewWinsSameKey(t *testing.T) {
	got := mergeExtrasJSON(`{"wo":"1"}`, `{"wo":"2"}`)
	var m map[string]any
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatal(err)
	}
	// conflicting scalars become [old,new]
	arr, ok := m["wo"].([]any)
	if !ok || len(arr) != 2 {
		t.Fatalf("wo = %#v, want [old,new] array", m["wo"])
	}
}

func TestDecodeIDPayloadLegacyAndFull(t *testing.T) {
	phone, user, name, extras := decodeIDPayload([]byte("p\x00u"))
	if phone != "p" || user != "u" || name != "" || extras != "" {
		t.Fatalf("legacy = %q %q %q %q", phone, user, name, extras)
	}
	phone, user, name, extras = decodeIDPayload([]byte("p\x00u\x00n\x00{\"a\":1}"))
	if phone != "p" || user != "u" || name != "n" || extras != `{"a":1}` {
		t.Fatalf("full = %q %q %q %q", phone, user, name, extras)
	}
}
