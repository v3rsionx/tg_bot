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
	"github.com/v3rsionx/tg_bot/internal/search"
)

func cmdView(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id is required")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	db, err := openLMDBReadOnly(ctx, cfg.LMDBIDPath)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	printPathHint(cfg.LMDBIDPath)

	payload, err := db.Get(ctx, []byte(id))
	if err != nil {
		if errors.Is(err, lmdb.ErrNotFound) {
			fmt.Println("NOT FOUND")
			return nil
		}
		return err
	}
	rec, err := search.DecodeIDPayload(id, payload)
	if err != nil {
		return err
	}
	printRecord(rec)
	return nil
}

func cmdStats() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()

	idDB, err := openLMDBReadOnly(ctx, cfg.LMDBIDPath)
	if err != nil {
		return err
	}
	defer func() { _ = idDB.Close() }()

	phoneDB, err := openLMDBReadOnly(ctx, cfg.LMDBPhonePath)
	if err != nil {
		return err
	}
	defer func() { _ = phoneDB.Close() }()

	userDB, err := openLMDBReadOnly(ctx, cfg.LMDBUsernamePath)
	if err != nil {
		return err
	}
	defer func() { _ = userDB.Close() }()

	idStats, err := idDB.Stats(ctx)
	if err != nil {
		return fmt.Errorf("id stats: %w", err)
	}
	phoneStats, err := phoneDB.Stats(ctx)
	if err != nil {
		return fmt.Errorf("phone stats: %w", err)
	}
	userStats, err := userDB.Stats(ctx)
	if err != nil {
		return fmt.Errorf("username stats: %w", err)
	}

	printPathHint(cfg.LMDBIDPath)
	fmt.Printf("id_entries=%d\n", idStats.Entries)
	fmt.Printf("phone_entries=%d\n", phoneStats.Entries)
	fmt.Printf("username_entries=%d\n", userStats.Entries)
	return nil
}

func cmdSample(n int) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	db, err := openLMDBReadOnly(ctx, cfg.LMDBIDPath)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	printPathHint(cfg.LMDBIDPath)

	txn, err := db.Reader(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = txn.Abort() }()

	cur, err := txn.Cursor(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = cur.Close() }()

	key, value, err := cur.First(ctx)
	shown := 0
	for err == nil && shown < n {
		rec, decErr := search.DecodeIDPayload(string(key), value)
		if decErr != nil {
			return decErr
		}
		if shown > 0 {
			fmt.Println("---")
		}
		printRecord(rec)
		shown++
		key, value, err = cur.Next(ctx)
	}
	if err != nil && !errors.Is(err, lmdb.ErrNotFound) {
		return err
	}
	if shown == 0 {
		fmt.Println("EMPTY")
		return nil
	}
	fmt.Printf("\nsampled=%d\n", shown)
	return nil
}

func openLMDBReadOnly(ctx context.Context, path string) (*lmdb.DB, error) {
	db, err := lmdb.OpenDB(ctx, lmdb.Config{Path: path, ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("open lmdb %q: %w", path, err)
	}
	return db, nil
}

func printPathHint(path string) {
	if abs, err := filepath.Abs(path); err == nil {
		fmt.Fprintf(os.Stderr, "lmdb: %s\n", abs)
	}
}

func printRecord(rec search.Record) {
	extras := rec.Extras
	if extras == "{}" {
		extras = ""
	}
	fmt.Printf("ID: %s\n", rec.ID)
	fmt.Printf("Name: %s\n", rec.Name)
	fmt.Printf("Phone: %s\n", rec.Phone)
	fmt.Printf("Username: %s\n", rec.Username)
	fmt.Printf("Extras: %s\n", extras)
}
