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
}

// TestValidatorRejectsInvalidID ensures malformed IDs are skipped by callers.
func TestValidatorRejectsInvalidID(t *testing.T) {
	v := NewValidator(Config{IDColumn: 0, PhoneColumn: 1, UsernameColumn: 2}.withDefaults())
	if _, err := v.ValidateFields([]string{"0x1", "+15551234567", "alice"}, Record{}); err == nil {
		t.Fatal("ValidateFields() error = nil, want error")
	}
}
