// Command importer imports CSV/TXT search data into LMDB indexes.
//
// Examples:
//
//	go run ./cmd/importer -file dumps/users.csv -resume -header
//	go run ./cmd/importer -dir dumps/ -resume
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	var (
		filesFlag      = flag.String("file", "", "comma-separated CSV/TXT file paths")
		dirFlag        = flag.String("dir", "", "comma-separated folders to scan for CSV/TXT")
		resumeFlag     = flag.Bool("resume", false, "resume from checkpoint")
		headerFlag     = flag.Bool("header", false, "skip CSV/TXT header row")
		delimiterFlag  = flag.String("delimiter", ",", "field delimiter")
		checkpointFlag = flag.String("checkpoint", "", "checkpoint file path (default when -resume)")
	)
	flag.Parse()

	files := splitCSV(*filesFlag)
	files = append(files, flag.Args()...)
	dirs := splitCSV(*dirFlag)

	opts := options{
		Files:      files,
		Dirs:       dirs,
		Resume:     *resumeFlag,
		HasHeader:  *headerFlag,
		Delimiter:  *delimiterFlag,
		Checkpoint: *checkpointFlag,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	application, err := buildImporterApp(ctx, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "importer startup failed: %v\n", err)
		os.Exit(1)
	}
	defer application.shutdown()

	if err := application.run(ctx); err != nil && ctx.Err() == nil {
		fmt.Fprintf(os.Stderr, "importer exited with error: %v\n", err)
		os.Exit(1)
	}
}

func splitCSV(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
