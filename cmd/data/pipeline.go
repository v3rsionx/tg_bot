package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/v3rsionx/tg_bot/internal/config"
	"github.com/v3rsionx/tg_bot/internal/converter"
	"github.com/v3rsionx/tg_bot/internal/database/lmdb"
	"github.com/v3rsionx/tg_bot/internal/importer"
)

func cmdConvert(paths []string) error {
	sources, err := resolveConvertInputs(paths)
	if err != nil {
		return err
	}
	ctx := context.Background()
	results, err := runConverter(ctx, sources, false)
	if err != nil {
		return err
	}
	for _, res := range results {
		fmt.Println("========== CONVERT ==========")
		fmt.Println(converter.FormatSummary(res.OutputFile, res.Statistics))
		fmt.Printf("output=%s\n", res.OutputFile)
	}
	return nil
}

func cmdImport(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("csv path is required")
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}
	stats, err := runImport(context.Background(), []string{path})
	if err != nil {
		return err
	}
	fmt.Println("========== IMPORT ==========")
	fmt.Println(importer.FormatStatistics(stats))
	return nil
}

func cmdAdd(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("file path is required")
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}

	importPath := path
	standard, err := hasStandardHeader(path)
	if err != nil {
		return err
	}
	if !standard && !strings.HasSuffix(strings.ToLower(path), ".standard.csv") {
		fmt.Fprintf(os.Stderr, "converting %s ...\n", path)
		results, convErr := runConverter(context.Background(), []string{path}, false)
		if convErr != nil {
			return convErr
		}
		if len(results) == 0 {
			return fmt.Errorf("converter produced no output for %q", path)
		}
		importPath = results[0].OutputFile
		fmt.Fprintf(os.Stderr, "converted -> %s\n", importPath)
	} else {
		fmt.Fprintf(os.Stderr, "standard CSV detected, importing directly\n")
	}

	stats, err := runImport(context.Background(), []string{importPath})
	if err != nil {
		return err
	}
	fmt.Println("========== ADD ==========")
	fmt.Printf("source=%s\n", importPath)
	fmt.Println(importer.FormatStatistics(stats))
	return nil
}

func runConverter(ctx context.Context, sources []string, dryRun bool) ([]converter.Result, error) {
	log := stdioLogger{}
	cfg := converter.Config{
		Sources: sources,
		DryRun:  dryRun,
		LogPath: "logs/converter.log",
	}
	c, err := converter.New(cfg, log, func(p converter.Progress) {
		fmt.Fprintf(os.Stderr, "\r%s", converter.FormatProgress(p))
	})
	if err != nil {
		return nil, err
	}
	defer c.Close()

	results, _, err := c.Run(ctx)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func runImport(ctx context.Context, sources []string) (importer.Statistics, error) {
	cfg, err := config.Load()
	if err != nil {
		return importer.Statistics{}, err
	}

	// Large dumps need a bigger starting map; auto-growth still applies.
	const largeMapSize int64 = 16 << 30 // 16 GiB

	idDB, err := lmdb.OpenDB(ctx, lmdb.Config{
		Path:           cfg.LMDBIDPath,
		InitialMapSize: largeMapSize,
	})
	if err != nil {
		return importer.Statistics{}, fmt.Errorf("open lmdb id: %w", err)
	}
	defer func() { _ = idDB.Close() }()

	phoneDB, err := lmdb.OpenDB(ctx, lmdb.Config{
		Path:           cfg.LMDBPhonePath,
		InitialMapSize: largeMapSize,
	})
	if err != nil {
		return importer.Statistics{}, fmt.Errorf("open lmdb phone: %w", err)
	}
	defer func() { _ = phoneDB.Close() }()

	userDB, err := lmdb.OpenDB(ctx, lmdb.Config{
		Path:           cfg.LMDBUsernamePath,
		InitialMapSize: largeMapSize,
	})
	if err != nil {
		return importer.Statistics{}, fmt.Errorf("open lmdb username: %w", err)
	}
	defer func() { _ = userDB.Close() }()

	impCfg := importer.Config{
		Sources:          append([]string(nil), sources...),
		Delimiter:        ',',
		Workers:          cfg.WorkerCount,
		BatchSize:        cfg.BatchSize,
		UpdateExisting:   true,
		SkipDuplicateIDs: false,
		Resume:           true,
		CheckpointPath:   "data/importer.checkpoint.json",
		ProgressInterval: 2 * time.Second,
	}.WithAutoMapHeaders(true)

	imp, err := importer.New(impCfg, importer.Stores{
		ID:       idDB,
		Phone:    phoneDB,
		Username: userDB,
	}, stdioLogger{}, func(p importer.Progress) {
		fmt.Fprintf(os.Stderr, "\r%s", importer.FormatStatistics(p.Statistics))
	})
	if err != nil {
		return importer.Statistics{}, err
	}

	printPathHint(cfg.LMDBIDPath)
	stats, err := imp.Run(ctx)
	fmt.Fprintln(os.Stderr)
	return stats, err
}

func hasStandardHeader(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()

	reader := csv.NewReader(bufio.NewReader(f))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	fields, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return false, nil
		}
		return false, err
	}
	seen := map[string]bool{}
	for _, field := range fields {
		key := normalizeHeaderToken(field)
		if key != "" {
			seen[key] = true
		}
	}
	return seen["id"] && seen["name"] && seen["phone"] && seen["username"] && seen["extras"], nil
}

func normalizeHeaderToken(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	raw = strings.TrimPrefix(raw, "\ufeff")
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "")
	raw = replacer.Replace(raw)
	switch raw {
	case "id", "userid", "telegramid", "tgid":
		return "id"
	case "name", "fullname":
		return "name"
	case "phone", "phonenumber", "mobile", "number":
		return "phone"
	case "username", "user", "uname", "handle":
		return "username"
	case "extras", "extra", "meta", "metadata", "logs", "log":
		return "extras"
	default:
		return raw
	}
}

func resolveConvertInputs(args []string) ([]string, error) {
	seen := map[string]struct{}{}
	var out []string
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg == "" {
			continue
		}
		info, err := os.Stat(arg)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			entries, err := os.ReadDir(arg)
			if err != nil {
				return nil, err
			}
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				name := e.Name()
				ext := strings.ToLower(filepath.Ext(name))
				if ext != ".csv" && ext != ".txt" {
					continue
				}
				if strings.HasSuffix(strings.ToLower(name), ".standard.csv") {
					continue
				}
				path := filepath.Join(arg, name)
				abs, _ := filepath.Abs(path)
				if _, ok := seen[abs]; ok {
					continue
				}
				seen[abs] = struct{}{}
				out = append(out, path)
			}
			continue
		}
		ext := strings.ToLower(filepath.Ext(arg))
		if ext != ".csv" && ext != ".txt" {
			return nil, fmt.Errorf("unsupported input %q (want .csv or .txt)", arg)
		}
		abs, _ := filepath.Abs(arg)
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		out = append(out, arg)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no CSV/TXT sources found")
	}
	return out, nil
}

type stdioLogger struct{}

func (stdioLogger) Debugf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "DEBUG "+format+"\n", args...)
}
func (stdioLogger) Infof(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "INFO "+format+"\n", args...)
}
func (stdioLogger) Warnf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "WARN "+format+"\n", args...)
}
func (stdioLogger) Errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ERROR "+format+"\n", args...)
}

var (
	_ importer.Logger  = stdioLogger{}
	_ converter.Logger = stdioLogger{}
)
