package search

import "testing"

// TestNormalizePhoneExactMatchRules verifies phone normalization for exact keys.
func TestNormalizePhoneExactMatchRules(t *testing.T) {
	got, err := normalizePhone("+1 (555) 111-0001")
	if err != nil {
		t.Fatalf("normalizePhone() error = %v", err)
	}
	if got != "+15551110001" {
		t.Fatalf("normalizePhone() = %q, want +15551110001", got)
	}
}

// TestNormalizeUsernameExactMatchRules verifies username normalization.
func TestNormalizeUsernameExactMatchRules(t *testing.T) {
	got, err := normalizeUsername("@Alice_One")
	if err != nil {
		t.Fatalf("normalizeUsername() error = %v", err)
	}
	if got != "alice_one" {
		t.Fatalf("normalizeUsername() = %q, want alice_one", got)
	}
}

// TestDecodeIDPayloadRoundTrip verifies importer-compatible payload decoding.
func TestDecodeIDPayloadRoundTrip(t *testing.T) {
	payload := append(append([]byte("+15551110001"), 0), []byte("alice_one")...)
	record, err := decodeIDPayload("1001", payload)
	if err != nil {
		t.Fatalf("decodeIDPayload() error = %v", err)
	}
	if record.ID != "1001" || record.Phone != "+15551110001" || record.Username != "alice_one" {
		t.Fatalf("unexpected record: %+v", record)
	}
}
