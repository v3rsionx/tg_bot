package converter

import (
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
)

func TestConvertJSONLFileMapsAdapterUserID(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "spider.jsonl")
	content := `{"nick":"Алиме","adapterType":"telegram","adapterUserId":"942174538","name":"Алиме","firstName":"Алиме","lastName":"","phone":"","id":6585,"customerId":70965}
{"nick":"Tanushka_y","adapterUserId":"726163395","name":"Tanushka_y","phone":"15551234567","id":6642}
{"nick":"bad","adapterUserId":"","name":"x","id":1}
`
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := ConvertJSONLFile(context.Background(), src)
	if err != nil {
		t.Fatalf("ConvertJSONLFile: %v", err)
	}
	if res.Statistics.OutputRows != 2 {
		t.Fatalf("OutputRows = %d, want 2", res.Statistics.OutputRows)
	}
	if res.Statistics.SkippedRows != 1 {
		t.Fatalf("SkippedRows = %d, want 1", res.Statistics.SkippedRows)
	}

	f, err := os.Open(res.OutputFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 { // header + 2
		t.Fatalf("rows = %d, want 3", len(rows))
	}
	if rows[0][0] != "id" {
		t.Fatalf("header = %#v", rows[0])
	}
	// CRM id 6585 must NOT become Telegram id.
	if rows[1][0] != "942174538" || rows[1][1] != "Алиме" {
		t.Fatalf("row1 = %#v", rows[1])
	}
	if rows[1][3] != "" { // Cyrillic nick is not a telegram username
		t.Fatalf("username = %q, want empty", rows[1][3])
	}
	if rows[2][0] != "726163395" || rows[2][3] != "tanushka_y" {
		t.Fatalf("row2 = %#v", rows[2])
	}
}

func TestLooksLikeJSONL(t *testing.T) {
	dir := t.TempDir()
	jsonl := filepath.Join(dir, "a.jsonl")
	_ = os.WriteFile(jsonl, []byte(`{"adapterUserId":"1"}`+"\n"), 0o644)
	if !LooksLikeJSONL(jsonl) {
		t.Fatal("expected jsonl")
	}
	csvPath := filepath.Join(dir, "b.csv")
	_ = os.WriteFile(csvPath, []byte("id,name,phone,username,extras\n1,a,,,\n"), 0o644)
	if LooksLikeJSONL(csvPath) {
		t.Fatal("csv should not look like jsonl")
	}
}
