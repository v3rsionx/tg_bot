package converter

import (
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeAndClassify(t *testing.T) {
	cases := map[string]FieldRole{
		"Telegram_ID": RoleID,
		"user-id":     RoleID,
		"First Name":  RoleName,
		"last_name":   RoleLastName,
		"Mobile":      RolePhone,
		"NickName":    RoleUsername,
	}
	for in, want := range cases {
		got, ok := classifyHeader(in)
		if !ok || got != want {
			t.Fatalf("classifyHeader(%q) = %q ok=%v, want %q", in, got, ok, want)
		}
	}
}

func TestDetectDelimiterAndConvert(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "dump.csv")
	content := "telegram_id;first_name;surname;phone;username;access_hash;country\n" +
		"6473397867;Fabiana;Umbelino;;fabiana;81293;BR\n" +
		";NoID;User;123;x;1;US\n" +
		"100;Ana;Silva;5511999;ana;9;BR\n"
	if err := os.WriteFile(in, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(dir, "converter.log")
	cfg := Config{
		Sources:       []string{in},
		LogPath:       logPath,
		CheckpointDir: filepath.Join(dir, "cp"),
	}
	var last Progress
	c, err := New(cfg, NopLogger{}, func(p Progress) { last = p })
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	res, err := c.ConvertFile(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if res.Detection.Delimiter != ';' {
		t.Fatalf("delimiter = %q", string(res.Detection.Delimiter))
	}
	if res.Statistics.OutputRows != 2 {
		t.Fatalf("output rows = %d, want 2", res.Statistics.OutputRows)
	}
	if res.Statistics.SkippedRows != 1 {
		t.Fatalf("skipped = %d, want 1", res.Statistics.SkippedRows)
	}
	if last.Processed == 0 {
		t.Fatal("expected progress updates")
	}

	f, err := os.Open(res.OutputFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 { // header + 2
		t.Fatalf("rows = %d", len(rows))
	}
	if rows[0][0] != "id" || rows[0][4] != "extras" {
		t.Fatalf("header = %#v", rows[0])
	}
	if rows[1][0] != "6473397867" {
		t.Fatalf("id = %q", rows[1][0])
	}
	if !strings.Contains(rows[1][1], "Fabiana") || !strings.Contains(rows[1][1], "Umbelino") {
		t.Fatalf("name = %q", rows[1][1])
	}
	if !strings.Contains(rows[1][4], "access_hash") || !strings.Contains(rows[1][4], "81293") {
		t.Fatalf("extras = %q", rows[1][4])
	}

	skipRaw, _ := os.ReadFile(logPath)
	if !strings.Contains(string(skipRaw), "missing id") {
		t.Fatalf("skip log missing entry: %s", skipRaw)
	}
}

func TestDryRun(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "a.txt")
	content := "id\tusername\tfoo\n1\tuser\tbar\n"
	if err := os.WriteFile(in, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := New(Config{
		Sources:       []string{in},
		DryRun:        true,
		LogPath:       filepath.Join(dir, "c.log"),
		CheckpointDir: filepath.Join(dir, "cp"),
	}, NopLogger{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	rep, err := c.DryRun(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Detection.Delimiter != '\t' {
		t.Fatalf("delimiter = %q", string(rep.Detection.Delimiter))
	}
	if len(rep.ExtrasKeys) != 1 || rep.ExtrasKeys[0] != "foo" {
		t.Fatalf("extras = %#v", rep.ExtrasKeys)
	}
	out := standardOutputPath(in)
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("dry-run must not create output")
	}
}
