package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/v3rsionx/tg_bot/internal/config"
	"github.com/v3rsionx/tg_bot/internal/constants"
	"github.com/v3rsionx/tg_bot/internal/database/lmdb"
	"github.com/v3rsionx/tg_bot/internal/importer"
	applogger "github.com/v3rsionx/tg_bot/internal/logger"
	"github.com/v3rsionx/tg_bot/internal/metrics"
)

// options holds importer CLI options.
type options struct {
	Files      []string
	Dirs       []string
	Resume     bool
	HasHeader  bool
	Delimiter  string
	Checkpoint string
}

// app owns the wired importer process graph.
type app struct {
	cfg      *config.Config
	opts     options
	sources  []string
	log      *applogger.Base
	metrics  *metrics.Collector
	lmdbID   *lmdb.DB
	lmdbPhone *lmdb.DB
	lmdbUser  *lmdb.DB
	importer *importer.Importer
}

func buildImporterApp(ctx context.Context, opts options) (*app, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load configuration: %w", err)
	}

	sources, err := resolveSources(opts.Files, opts.Dirs)
	if err != nil {
		return nil, fmt.Errorf("resolve sources: %w", err)
	}

	a := &app{cfg: cfg, opts: opts, sources: sources}

	level := applogger.ParseLevel(cfg.LogLevel)
	console := applogger.NewConsoleSink(os.Stdout)
	fileSink, err := applogger.NewFileSink(applogger.RotateOptions{
		Filename:   "logs/importer.log",
		MaxSizeMB:  constants.DefaultLogMaxSizeMB,
		MaxBackups: constants.DefaultLogMaxBackups,
		MaxAgeDays: constants.DefaultLogMaxAgeDays,
		Compress:   true,
		Daily:      true,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize file logger: %w", err)
	}
	a.log = applogger.NewMultiLogger(level, console, fileSink)
	a.log.Info("configuration loaded", applogger.Fields{
		"version": cfg.Version,
		"sources": len(sources),
	})

	a.metrics = metrics.New()
	a.log.Info("metrics initialized")

	idDB, err := lmdb.OpenDB(ctx, lmdb.Config{Path: cfg.LMDBIDPath})
	if err != nil {
		_ = a.log.Close()
		return nil, fmt.Errorf("open lmdb id: %w", err)
	}
	a.lmdbID = idDB

	phoneDB, err := lmdb.OpenDB(ctx, lmdb.Config{Path: cfg.LMDBPhonePath})
	if err != nil {
		a.partialClose()
		return nil, fmt.Errorf("open lmdb phone: %w", err)
	}
	a.lmdbPhone = phoneDB

	userDB, err := lmdb.OpenDB(ctx, lmdb.Config{Path: cfg.LMDBUsernamePath})
	if err != nil {
		a.partialClose()
		return nil, fmt.Errorf("open lmdb username: %w", err)
	}
	a.lmdbUser = userDB
	a.log.Info("LMDB Connected", applogger.Fields{
		"id":       cfg.LMDBIDPath,
		"phone":    cfg.LMDBPhonePath,
		"username": cfg.LMDBUsernamePath,
	})

	delimiter := ','
	if opts.Delimiter != "" {
		r := []rune(opts.Delimiter)
		if len(r) != 1 {
			a.partialClose()
			return nil, fmt.Errorf("delimiter must be a single character")
		}
		delimiter = r[0]
	}

	checkpoint := opts.Checkpoint
	if checkpoint == "" && opts.Resume {
		checkpoint = "data/importer.checkpoint.json"
	}

	portLogger := newPrintfLogger(a.log.With(applogger.Fields{"component": "importer"}))
	imp, err := importer.New(importer.Config{
		Sources:          append([]string(nil), sources...),
		Delimiter:        delimiter,
		HasHeader:        opts.HasHeader,
		Workers:          cfg.WorkerCount,
		BatchSize:        cfg.BatchSize,
		CheckpointPath:   checkpoint,
		Resume:           opts.Resume,
		SkipDuplicateIDs: true,
		ProgressInterval: 2 * time.Second,
	}, importer.Stores{
		ID:       a.lmdbID,
		Phone:    a.lmdbPhone,
		Username: a.lmdbUser,
	}, portLogger, a.onProgress)
	if err != nil {
		a.partialClose()
		return nil, fmt.Errorf("initialize importer: %w", err)
	}
	a.importer = imp
	a.log.Info("importer initialized", applogger.Fields{
		"resume":     opts.Resume,
		"has_header": opts.HasHeader,
		"workers":    cfg.WorkerCount,
		"batch_size": cfg.BatchSize,
	})

	return a, nil
}

func (a *app) onProgress(p importer.Progress) {
	a.log.Info("import progress", applogger.Fields{
		"file":    p.CurrentFile,
		"summary": importer.FormatStatistics(p.Statistics),
	})
}

func (a *app) logStartupBanner() {
	a.log.Info("startup", applogger.Fields{
		"go_version":      runtime.Version(),
		"project_version": Version,
		"git_commit":      GitCommit,
		"build_time":      BuildTime,
	})
}

func (a *app) run(ctx context.Context) error {
	a.logStartupBanner()

	fmt.Println("READY")
	a.log.Info("READY")

	started := time.Now()
	stats, err := a.importer.Run(ctx)
	elapsed := time.Since(started)
	a.metrics.ObserveImport(int64(stats.Inserts+stats.Updates), elapsed)

	fmt.Println("SUMMARY")
	fmt.Println(importer.FormatStatistics(stats))
	a.log.Info("import summary", applogger.Fields{
		"summary":    importer.FormatStatistics(stats),
		"elapsed":    elapsed.String(),
		"inserts":    stats.Inserts,
		"updates":    stats.Updates,
		"duplicates": stats.Duplicates,
		"invalid":    stats.RecordsInvalid,
		"rps":        stats.RecordsPerSecond,
	})

	snap := a.metrics.Snapshot()
	a.log.Info("statistics", applogger.Fields{
		"import_speed_rps": snap.ImportSpeedRPS,
		"memory_alloc":     snap.MemoryAllocBytes,
		"cpu_count":        snap.CPUCount,
	})

	return err
}

func (a *app) shutdown() {
	if a == nil {
		return
	}
	if a.log != nil {
		a.log.Info("graceful shutdown started")
	}
	if a.lmdbUser != nil {
		_ = a.lmdbUser.Close()
	}
	if a.lmdbPhone != nil {
		_ = a.lmdbPhone.Close()
	}
	if a.lmdbID != nil {
		_ = a.lmdbID.Close()
	}
	if a.metrics != nil && a.log != nil {
		snap := a.metrics.Snapshot()
		a.log.Info("metrics snapshot", applogger.Fields{
			"import_speed_rps": snap.ImportSpeedRPS,
			"memory_alloc":     snap.MemoryAllocBytes,
		})
	}
	if a.log != nil {
		a.log.Info("graceful shutdown complete")
		_ = a.log.Close()
	}
}

func (a *app) partialClose() {
	if a == nil {
		return
	}
	if a.lmdbUser != nil {
		_ = a.lmdbUser.Close()
		a.lmdbUser = nil
	}
	if a.lmdbPhone != nil {
		_ = a.lmdbPhone.Close()
		a.lmdbPhone = nil
	}
	if a.lmdbID != nil {
		_ = a.lmdbID.Close()
		a.lmdbID = nil
	}
	if a.log != nil {
		_ = a.log.Close()
		a.log = nil
	}
}
