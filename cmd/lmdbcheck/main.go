// Command lmdbcheck is a temporary tool to verify an ID key in the LMDB ID store.
//
// Usage:
//
//	lmdbcheck.exe <id>
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/v3rsionx/tg_bot/internal/config"
	"github.com/v3rsionx/tg_bot/internal/database/lmdb"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <id>\n", filepath.Base(os.Args[0]))
		os.Exit(2)
	}
	id := strings.TrimSpace(os.Args[1])
	if id == "" {
		fmt.Fprintf(os.Stderr, "usage: %s <id>\n", filepath.Base(os.Args[0]))
		os.Exit(2)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	db, err := lmdb.OpenDB(ctx, lmdb.Config{
		Path:     cfg.LMDBIDPath,
		ReadOnly: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "open lmdb id %q: %v\n", cfg.LMDBIDPath, err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	if abs, err := filepath.Abs(cfg.LMDBIDPath); err == nil {
		fmt.Fprintf(os.Stderr, "lmdb id path: %s\n", abs)
	}

	payload, err := db.Get(ctx, []byte(id))
	if err != nil {
		if errors.Is(err, lmdb.ErrNotFound) {
			fmt.Println("NOT FOUND")
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "get: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("FOUND")
	fmt.Printf("payload_len=%d\n", len(payload))
	fmt.Printf("payload_hex=%x\n", payload)
}
