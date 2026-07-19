package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/v3rsionx/tg_bot/internal/cache"
	"github.com/v3rsionx/tg_bot/internal/config"
	"github.com/v3rsionx/tg_bot/internal/constants"
	"github.com/v3rsionx/tg_bot/internal/database/lmdb"
	"github.com/v3rsionx/tg_bot/internal/database/sqlite"
	applogger "github.com/v3rsionx/tg_bot/internal/logger"
	"github.com/v3rsionx/tg_bot/internal/metrics"
	"github.com/v3rsionx/tg_bot/internal/search"
	"github.com/v3rsionx/tg_bot/internal/service"
	"github.com/v3rsionx/tg_bot/internal/telegram"
)

// app owns the fully wired bot process graph.
type app struct {
	cfg *config.Config

	log      *applogger.Base
	metrics  *metrics.Collector
	searchCache *cache.SearchCache
	userCache   *cache.UserCache
	adminCache  *cache.AdminCache

	sqliteDB *sqlite.DatabaseManager
	lmdbID   *lmdb.DB
	lmdbPhone *lmdb.DB
	lmdbUser  *lmdb.DB

	searchEngine *search.Service
	module       *service.Module
	bot          *telegram.Bot
}

// buildApp loads configuration and wires every module in startup order.
func buildApp(ctx context.Context) (*app, error) {
	// 1) Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load configuration: %w", err)
	}

	a := &app{cfg: cfg}

	// 2) Initialize logger
	level := applogger.ParseLevel(cfg.LogLevel)
	console := applogger.NewConsoleSink(os.Stdout)
	fileSink, err := applogger.NewFileSink(applogger.RotateOptions{
		Filename:   constants.DefaultLogFileName,
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
		"version":   cfg.Version,
		"log_level": cfg.LogLevel,
	})

	// 3) Initialize metrics
	a.metrics = metrics.New()
	a.log.Info("metrics initialized")

	// 4) Initialize cache
	a.searchCache = cache.NewSearchCache(cfg.CacheTTL, constants.DefaultSearchCacheSize)
	a.userCache = cache.NewUserCache(cfg.CacheTTL, constants.DefaultUserCacheSize)
	a.adminCache = cache.NewAdminCache(cfg.CacheTTL, constants.DefaultAdminCacheSize)
	a.log.Info("cache initialized", applogger.Fields{
		"cache_ttl": cfg.CacheTTL.String(),
	})

	// 5) Open SQLite
	sqlDB, err := sqlite.OpenFromConfig(cfg)
	if err != nil {
		_ = a.log.Close()
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	a.sqliteDB = sqlDB
	a.log.Info("SQLite Connected", applogger.Fields{"path": cfg.SQLitePath})

	// 6) Run SQLite migration (OpenFromConfig already migrates; re-run is idempotent)
	if err := a.sqliteDB.Migrate(ctx); err != nil {
		a.partialClose()
		return nil, fmt.Errorf("sqlite migration: %w", err)
	}
	a.log.Info("sqlite migrations applied")

	// 7) Open LMDB indexes
	idDB, err := lmdb.OpenDB(ctx, lmdb.Config{Path: cfg.LMDBIDPath})
	if err != nil {
		a.partialClose()
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
	// Temporary debug logging — remove after LMDB path/payload investigation.
	if idAbs, err := filepath.Abs(cfg.LMDBIDPath); err != nil {
		a.log.Warn("LMDB ID path abs resolve failed", applogger.Fields{
			"id":  cfg.LMDBIDPath,
			"err": err.Error(),
		})
	} else {
		a.log.Info("LMDB ID directory (absolute)", applogger.Fields{"id_abs": idAbs})
	}

	// 8) Initialize search engine
	engine, err := search.New(search.Config{
		Timeout:  cfg.SearchTimeout,
		CacheTTL: cfg.CacheTTL,
	}, search.Stores{
		ID:       a.lmdbID,
		Phone:    a.lmdbPhone,
		Username: a.lmdbUser,
	})
	if err != nil {
		a.partialClose()
		return nil, fmt.Errorf("initialize search engine: %w", err)
	}
	a.searchEngine = engine
	a.log.Info("search engine initialized")

	// 9) Initialize services
	repos := a.sqliteDB.Repositories()
	portLogger := newPrintfLogger(a.log.With(applogger.Fields{"component": "service"}))
	module, err := service.NewModule(service.Config{
		OwnerIDs:            append([]int64(nil), cfg.BotOwnerIDs...),
		PointsPerSearch:     cfg.PointsPerSearch,
		SearchRateLimit:     cfg.RateLimit,
		SearchRateWindow:    time.Minute,
		DefaultHistoryLimit: constants.DefaultHistoryLimit,
	}, service.ModuleDeps{
		Users:        repos.Users,
		Transactions: repos.Transactions,
		History:      repos.SearchHistory,
		Engine:       a.searchEngine,
		Logger:       portLogger,
	})
	if err != nil {
		a.partialClose()
		return nil, fmt.Errorf("initialize services: %w", err)
	}
	a.module = module
	a.log.Info("services initialized")

	// 10) Initialize Telegram + 11) Register handlers (done inside telegram.New)
	tgLogger := newPrintfLogger(a.log.With(applogger.Fields{"component": "telegram"}))
	deps := service.TelegramDependencies(module, tgLogger)
	bot, err := telegram.New(telegram.Config{
		Token:           cfg.BotToken,
		OwnerIDs:        append([]int64(nil), cfg.BotOwnerIDs...),
		ShutdownTimeout: constants.DefaultShutdownWait,
		HistoryLimit:    constants.DefaultHistoryLimit,
	}, deps)
	if err != nil {
		a.partialClose()
		return nil, fmt.Errorf("initialize telegram: %w", err)
	}
	a.bot = bot
	a.log.Info("Bot Started", applogger.Fields{
		"owners": len(cfg.BotOwnerIDs),
	})

	return a, nil
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

	health := runHealthChecks(ctx, a.cfg, a.sqliteDB, a.lmdbID, a.lmdbPhone, a.lmdbUser)
	if !health.OK {
		return fmt.Errorf("health check failed: %s", health.Error())
	}
	a.log.Info("health check passed")

	// Readiness: print only when everything is initialized successfully.
	fmt.Println("READY")
	a.log.Info("READY")

	return a.bot.Start(ctx)
}

func (a *app) shutdown(parent context.Context) {
	if a == nil {
		return
	}
	ctx, cancel := context.WithTimeout(parent, constants.DefaultShutdownWait)
	defer cancel()

	if a.log != nil {
		a.log.Info("graceful shutdown started")
	}

	if a.bot != nil {
		if err := a.bot.Shutdown(ctx); err != nil && a.log != nil {
			a.log.Error("telegram shutdown failed", applogger.Fields{"error": err.Error()})
		}
	}
	if a.searchEngine != nil {
		if err := a.searchEngine.Close(); err != nil && a.log != nil {
			a.log.Error("search engine close failed", applogger.Fields{"error": err.Error()})
		}
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
	if a.sqliteDB != nil {
		if err := a.sqliteDB.Close(); err != nil && a.log != nil {
			a.log.Error("sqlite close failed", applogger.Fields{"error": err.Error()})
		}
	}
	if a.searchCache != nil {
		a.searchCache.InvalidateAll()
	}
	if a.userCache != nil {
		a.userCache.InvalidateAll()
	}
	if a.adminCache != nil {
		a.adminCache.InvalidateAll()
	}
	if a.metrics != nil && a.log != nil {
		snap := a.metrics.Snapshot()
		a.log.Info("metrics snapshot", applogger.Fields{
			"total_searches": snap.TotalSearches,
			"cache_hits":     snap.CacheHits,
			"cache_misses":   snap.CacheMisses,
			"memory_alloc":   snap.MemoryAllocBytes,
		})
	}
	if a.log != nil {
		a.log.Info("graceful shutdown complete")
		_ = a.log.Close()
	}
}

// partialClose releases resources acquired during a failed buildApp.
func (a *app) partialClose() {
	if a == nil {
		return
	}
	if a.searchEngine != nil {
		_ = a.searchEngine.Close()
		a.searchEngine = nil
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
	if a.sqliteDB != nil {
		_ = a.sqliteDB.Close()
		a.sqliteDB = nil
	}
	if a.log != nil {
		_ = a.log.Close()
		a.log = nil
	}
}
