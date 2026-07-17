package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSourcesFilesAndDir(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "a.csv")
	txtPath := filepath.Join(dir, "b.txt")
	other := filepath.Join(dir, "ignore.json")
	if err := os.WriteFile(csvPath, []byte("1,2,3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(txtPath, []byte("1\t2\t3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := resolveSources([]string{csvPath}, []string{dir})
	if err != nil {
		t.Fatalf("resolveSources: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (deduped file+dir)", len(got))
	}
}
