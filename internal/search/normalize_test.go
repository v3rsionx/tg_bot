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
	tests := []struct {
		name     string
		id       string
		payload  []byte
		want     Record
	}{
		{
			name:    "id only",
			id:      "6473397867",
			payload: []byte{0},
			want:    Record{ID: "6473397867"},
		},
		{
			name:    "id + phone",
			id:      "1001",
			payload: append([]byte("+15551110001"), 0),
			want:    Record{ID: "1001", Phone: "+15551110001"},
		},
		{
			name:    "id + username",
			id:      "1002",
			payload: append([]byte{0}, []byte("alice_one")...),
			want:    Record{ID: "1002", Username: "alice_one"},
		},
		{
			name:    "id + phone + username",
			id:      "1003",
			payload: append(append([]byte("+15551110003"), 0), []byte("bob")...),
			want:    Record{ID: "1003", Phone: "+15551110003", Username: "bob"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeIDPayload(tt.id, tt.payload)
			if err != nil {
				t.Fatalf("decodeIDPayload() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
