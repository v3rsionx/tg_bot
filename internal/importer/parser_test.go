package importer

import "testing"

// TestParseFieldsSupportsQuotedCSV verifies delimiter and quote handling.
func TestParseFieldsSupportsQuotedCSV(t *testing.T) {
	fields, err := parseFields(`123,"+1 (555) 000-1111","User_Name"`, ',')
	if err != nil {
		t.Fatalf("parseFields() error = %v", err)
	}
	if len(fields) != 3 {
		t.Fatalf("len(fields) = %d, want 3", len(fields))
	}
	if fields[0] != "123" || fields[2] != "User_Name" {
		t.Fatalf("unexpected fields: %#v", fields)
	}
}

// TestParseFieldsSupportsCustomDelimiter verifies TXT/TSV style input.
func TestParseFieldsSupportsCustomDelimiter(t *testing.T) {
	fields, err := parseFields("123\t+15550001111\talice_user", '\t')
	if err != nil {
		t.Fatalf("parseFields() error = %v", err)
	}
	if len(fields) != 3 || fields[1] != "+15550001111" {
		t.Fatalf("unexpected fields: %#v", fields)
	}
}
