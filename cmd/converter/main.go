// Command converter turns arbitrary CSV/TXT dumps into standard importer CSV.
//
// Examples:
//
//	converter input.csv
//	converter input.csv --dry-run
//	converter dumps/ --resume
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/v3rsionx/tg_bot/internal/converter"
)

func main() {
	var (
		dryRun = flag.Bool("dry-run", false, "detect mapping only (first 100 rows), do not write output")
		resume = flag.Bool("resume", false, "resume from checkpoint after interruption")
		logPath = flag.String("log", "logs/converter.log", "skipped-row log path")
	)
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: converter <file-or-dir> [more files/dirs...] [--dry-run] [--resume]\n")
		os.Exit(2)
	}

	sources, err := resolveInputs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "converter: %v\n", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log := &stdioLogger{}
	cfg := converter.Config{
		Sources: sources,
		DryRun:  *dryRun,
		Resume:  *resume,
		LogPath: *logPath,
	}

	var last converter.Progress
	c, err := converter.New(cfg, log, func(p converter.Progress) {
		last = p
		fmt.Fprintf(os.Stderr, "\r%s", converter.FormatProgress(p))
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "converter: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	results, reports, err := c.Run(ctx)
	fmt.Fprintln(os.Stderr)
	if err != nil && ctx.Err() == nil {
		fmt.Fprintf(os.Stderr, "converter: %v\n", err)
		os.Exit(1)
	}

	if *dryRun {
		for _, r := range reports {
			fmt.Println(converter.FormatDryRun(r))
		}
		return
	}

	for _, res := range results {
		fmt.Println("========== SUMMARY ==========")
		fmt.Println(converter.FormatSummary(res.OutputFile, res.Statistics))
		fmt.Printf("Encoding: %s\nDelimiter: %s\n", res.Detection.Encoding, res.Detection.DelimiterName)
		_ = last
	}
}

func resolveInputs(args []string) ([]string, error) {
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
				// Skip already-converted outputs.
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

func (stdioLogger) Infof(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "INFO "+format+"\n", args...)
}
func (stdioLogger) Warnf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "WARN "+format+"\n", args...)
}
func (stdioLogger) Errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ERROR "+format+"\n", args...)
}
