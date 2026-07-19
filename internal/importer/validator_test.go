package importer

import "testing"

// TestValidatorAcceptsNormalizedRecords verifies phone/username normalization.
func TestValidatorAcceptsNormalizedRecords(t *testing.T) {
	v := NewValidator(Config{IDColumn: 0, PhoneColumn: 1, UsernameColumn: 2}.withDefaults())
	record, err := v.ValidateFields([]string{"42", "+1 (555) 123-4567", "@Alice_User"}, Record{})
	if err != nil {
		t.Fatalf("ValidateFields() error = %v", err)
	}
	if record.Phone != "+15551234567" {
		t.Fatalf("Phone = %q, want +15551234567", record.Phone)
	}
	if record.Username != "alice_user" {
		t.Fatalf("Username = %q, want alice_user", record.Username)
	}
	if record.Extras != "" {
		t.Fatalf("Extras = %q, want empty for legacy layout", record.Extras)
	}
}

// TestValidatorRejectsInvalidID ensures malformed IDs are skipped by callers.
func TestValidatorRejectsInvalidID(t *testing.T) {
	v := NewValidator(Config{IDColumn: 0, PhoneColumn: 1, UsernameColumn: 2}.withDefaults())
	if _, err := v.ValidateFields([]string{"0x1", "+15551110001", "alice"}, Record{}); err == nil {
		t.Fatal("ValidateFields() error = nil, want error")
	}
}

// TestValidatorOptionalPhoneAndUsername covers accepted optional-field combinations.
func TestValidatorOptionalPhoneAndUsername(t *testing.T) {
	std := ColumnMapping{
		ID: 0, Name: 1, Phone: 2, Username: 3, Extras: 4, Source: "header:standard",
	}
	legacy := ColumnMapping{
		ID: 0, Name: unsetColumn, Phone: 1, Username: 2, Extras: unsetColumn, Source: "config",
	}

	tests := []struct {
		name    string
		mapping ColumnMapping
		fields  []string
		want    Record
	}{
		{
			name:    "id only",
			mapping: std,
			fields:  []string{"6473397867", "", "", "", ""},
			want:    Record{ID: "6473397867", Extras: "{}"},
		},
		{
			name:    "id + name",
			mapping: std,
			fields:  []string{"6473397867", "Fabiana Umbelino", "", "", ""},
			want:    Record{ID: "6473397867", Name: "Fabiana Umbelino", Extras: "{}"},
		},
		{
			name:    "id + extras",
			mapping: std,
			fields:  []string{"6473397867", "", "", "", `{"access_hash":"8129359283721321484"}`},
			want: Record{
				ID:     "6473397867",
				Extras: `{"access_hash":"8129359283721321484"}`,
			},
		},
		{
			name:    "id + phone",
			mapping: legacy,
			fields:  []string{"1001", "+1 (555) 111-0001", ""},
			want:    Record{ID: "1001", Phone: "+15551110001"},
		},
		{
			name:    "id + username",
			mapping: legacy,
			fields:  []string{"1002", "", "@Alice_User"},
			want:    Record{ID: "1002", Username: "alice_user"},
		},
		{
			name:    "id + phone + username",
			mapping: legacy,
			fields:  []string{"1003", "+15551110003", "bob"},
			want:    Record{ID: "1003", Phone: "+15551110003", Username: "bob"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator(Config{}.withDefaults())
			v.SetMapping(tt.mapping)
			got, err := v.ValidateFields(tt.fields, Record{})
			if err != nil {
				t.Fatalf("ValidateFields() error = %v", err)
			}
			if got.ID != tt.want.ID || got.Name != tt.want.Name ||
				got.Phone != tt.want.Phone || got.Username != tt.want.Username ||
				got.Extras != tt.want.Extras {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestValidatorDropsInvalidPhoneOrUsernameKeepsID imports dirty dump rows.
func TestValidatorDropsInvalidPhoneOrUsernameKeepsID(t *testing.T) {
	v := NewValidator(Config{IDColumn: 0, PhoneColumn: 1, UsernameColumn: 2}.withDefaults())

	got, err := v.ValidateFields([]string{"42", "12", "alice"}, Record{})
	if err != nil {
		t.Fatalf("invalid phone should be dropped, got err=%v", err)
	}
	if got.ID != "42" || got.Phone != "" || got.Username != "alice" {
		t.Fatalf("got %+v", got)
	}

	got, err = v.ValidateFields([]string{"42", "+15551110001", "ab"}, Record{})
	if err != nil {
		t.Fatalf("invalid username should be dropped, got err=%v", err)
	}
	if got.ID != "42" || got.Phone != "+15551110001" || got.Username != "" {
		t.Fatalf("got %+v", got)
	}

	got, err = v.ValidateFields([]string{"99", "12", "x!"}, Record{})
	if err != nil {
		t.Fatalf("both invalid should still keep id, got err=%v", err)
	}
	if got.ID != "99" || got.Phone != "" || got.Username != "" {
		t.Fatalf("got %+v", got)
	}

	got, err = v.ValidateFields([]string{"77", "+15551110077", "🔥PeRes_OK✨"}, Record{})
	if err != nil {
		t.Fatalf("emoji username should sanitize, got err=%v", err)
	}
	if got.Username != "peres_ok" {
		t.Fatalf("Username = %q, want peres_ok", got.Username)
	}
}
