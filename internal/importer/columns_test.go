package importer

import "testing"

func TestResolveHeaderMappingStandardAndLegacy(t *testing.T) {
	std, ok := resolveHeaderMapping([]string{"id", "name", "phone", "username", "extras"})
	if !ok {
		t.Fatal("expected standard header mapping")
	}
	if std.Source != "header:standard" || std.ID != 0 || std.Name != 1 || std.Phone != 2 || std.Username != 3 || std.Extras != 4 {
		t.Fatalf("standard mapping = %+v", std)
	}

	legacy, ok := resolveHeaderMapping([]string{"ID", "Phone", "User_Name"})
	if !ok {
		t.Fatal("expected legacy header mapping")
	}
	if legacy.Source != "header:legacy" || legacy.Phone != 1 || legacy.Username != 2 || legacy.Name != unsetColumn {
		t.Fatalf("legacy mapping = %+v", legacy)
	}

	if _, ok := resolveHeaderMapping([]string{"foo", "bar"}); ok {
		t.Fatal("expected non-header row to fail mapping")
	}
}

func TestValidatorStandardLayoutReadsExtrasAndName(t *testing.T) {
	v := NewValidator(Config{}.withDefaults())
	v.SetMapping(ColumnMapping{
		ID: 0, Name: 1, Phone: 2, Username: 3, Extras: 4, Source: "header:standard",
	})
	rec, err := v.ValidateFields([]string{
		"6473397867",
		"Fabiana Umbelino",
		"+15551234567",
		"fabiana",
		`{"access_hash":"81293","country":"BR"}`,
	}, Record{})
	if err != nil {
		t.Fatalf("ValidateFields: %v", err)
	}
	if rec.Name != "Fabiana Umbelino" {
		t.Fatalf("Name = %q", rec.Name)
	}
	if rec.Phone != "+15551234567" || rec.Username != "fabiana" {
		t.Fatalf("phone/user = %q %q", rec.Phone, rec.Username)
	}
	if rec.Extras != `{"access_hash":"81293","country":"BR"}` {
		t.Fatalf("Extras = %q", rec.Extras)
	}
}
