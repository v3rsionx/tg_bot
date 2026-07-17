package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// resolveSources expands CLI file/folder inputs into concrete CSV/TXT paths.
func resolveSources(files []string, dirs []string) ([]string, error) {
	seen := make(map[string]struct{})
	var out []string

	add := func(path string) error {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return fmt.Errorf("%s is a directory; use -dir", path)
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".csv" && ext != ".txt" {
			return fmt.Errorf("unsupported source type %q (want .csv or .txt)", path)
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			abs = path
		}
		if _, ok := seen[abs]; ok {
			return nil
		}
		seen[abs] = struct{}{}
		out = append(out, abs)
		return nil
	}

	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		if err := add(file); err != nil {
			return nil, err
		}
	}

	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("read dir %s: %w", dir, err)
		}
		var names []string
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext == ".csv" || ext == ".txt" {
				names = append(names, filepath.Join(dir, entry.Name()))
			}
		}
		sort.Strings(names)
		for _, name := range names {
			if err := add(name); err != nil {
				return nil, err
			}
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no CSV/TXT sources found")
	}
	return out, nil
}
