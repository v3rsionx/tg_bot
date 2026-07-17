package config

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/v3rsionx/tg_bot/internal/security"
	"github.com/v3rsionx/tg_bot/internal/validator"
)

// Options configures a Loader. Zero values select production defaults.
type Options struct {
	EnvFile       string
	JSONFile      string
	WatchInterval time.Duration
	// ApplyDotEnv copies missing .env keys into the process environment on Load/Reload.
	ApplyDotEnv bool
}

// ChangeHandler is invoked after a successful hot reload that changed values.
type ChangeHandler func(*Config)

// ErrorHandler is invoked when a hot-reload attempt fails validation or I/O.
type ErrorHandler func(error)

// Loader loads, validates, and optionally hot-reloads configuration.
// It is safe for concurrent use and holds no package-level mutable state.
type Loader struct {
	opts      Options
	validator validator.Validator
	sanitizer security.Sanitizer

	mu            sync.RWMutex
	current       *Config
	onChange      ChangeHandler
	onReloadError ErrorHandler
	lastError     error

	watchCancel context.CancelFunc
	watchWG     sync.WaitGroup
}

// NewLoader constructs a dependency-injected configuration loader.
func NewLoader(v validator.Validator, s security.Sanitizer, opts Options) *Loader {
	if v == nil {
		v = validator.New()
	}
	if s == nil {
		s = security.New()
	}
	if opts.EnvFile == "" {
		opts.EnvFile = defaultEnvFile
	}
	if opts.JSONFile == "" {
		opts.JSONFile = defaultJSONFile
	}
	if opts.WatchInterval <= 0 {
		opts.WatchInterval = defaultWatchInterval
	}
	return &Loader{
		opts:      opts,
		validator: v,
		sanitizer: s,
	}
}

// OnChange registers a handler invoked after each successful hot reload that
// produced a different configuration. Panics in the handler are recovered.
func (l *Loader) OnChange(handler ChangeHandler) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.onChange = handler
}

// OnReloadError registers a handler for failed hot-reload attempts.
func (l *Loader) OnReloadError(handler ErrorHandler) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.onReloadError = handler
}

// LastError returns the most recent hot-reload failure, if any.
func (l *Loader) LastError() error {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.lastError
}

// Load builds configuration from defaults, optional JSON, optional .env, and
// environment variable overrides, then validates the result.
func (l *Loader) Load() (*Config, error) {
	cfg, err := l.build()
	if err != nil {
		l.setLastError(err)
		return nil, err
	}
	l.mu.Lock()
	l.current = cfg.Clone()
	l.lastError = nil
	l.mu.Unlock()
	return cfg.Clone(), nil
}

// Get returns a copy of the last successfully loaded configuration.
func (l *Loader) Get() *Config {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.current.Clone()
}

// Reload rebuilds and validates configuration without process restart.
// On failure the previously loaded configuration is retained.
func (l *Loader) Reload() (*Config, error) {
	cfg, err := l.build()
	if err != nil {
		l.reportReloadError(err)
		return nil, err
	}

	l.mu.Lock()
	prev := l.current
	l.current = cfg.Clone()
	l.lastError = nil
	handler := l.onChange
	l.mu.Unlock()

	if handler != nil && !configsEqual(prev, cfg) {
		l.safeOnChange(handler, cfg.Clone())
	}
	return cfg.Clone(), nil
}

// StartWatch begins hot-reload polling of JSON and .env sources.
// It is a no-op when already watching. StopWatch cancels the loop.
func (l *Loader) StartWatch(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	l.mu.Lock()
	if l.watchCancel != nil {
		l.mu.Unlock()
		return
	}
	watchCtx, cancel := context.WithCancel(ctx)
	l.watchCancel = cancel
	l.mu.Unlock()

	l.watchWG.Add(1)
	go func() {
		defer l.watchWG.Done()
		l.watchLoop(watchCtx)
	}()
}

// StopWatch stops hot-reload polling and waits for the watcher to exit.
func (l *Loader) StopWatch() {
	l.mu.Lock()
	cancel := l.watchCancel
	l.watchCancel = nil
	l.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	l.watchWG.Wait()
}

func (l *Loader) watchLoop(ctx context.Context) {
	ticker := time.NewTicker(l.opts.WatchInterval)
	defer ticker.Stop()

	lastJSON := fileModTime(l.opts.JSONFile)
	lastEnv := fileModTime(l.opts.EnvFile)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			jsonMod := fileModTime(l.opts.JSONFile)
			envMod := fileModTime(l.opts.EnvFile)
			if !jsonMod.After(lastJSON) && !envMod.After(lastEnv) {
				continue
			}
			lastJSON = jsonMod
			lastEnv = envMod
			// Invalid reloads keep the last good config; errors go to OnReloadError.
			_, _ = l.Reload()
		}
	}
}

func (l *Loader) build() (*Config, error) {
	cfg := Defaults()

	doc, err := readJSONFile(l.opts.JSONFile)
	if err != nil {
		return nil, err
	}
	if err := doc.apply(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	envValues, err := readEnvFile(l.opts.EnvFile, l.sanitizer)
	if err != nil {
		return nil, err
	}
	if l.opts.ApplyDotEnv {
		if err := applyEnvMapToProcess(envValues); err != nil {
			return nil, err
		}
	}

	src := &valueSource{
		envFile:   envValues,
		sanitizer: l.sanitizer,
		validator: l.validator,
	}
	if err := src.applyOverrides(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	alignMaxResults(&cfg)
	if err := validateConfig(&cfg, l.validator, l.sanitizer); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (l *Loader) setLastError(err error) {
	l.mu.Lock()
	l.lastError = err
	l.mu.Unlock()
}

func (l *Loader) reportReloadError(err error) {
	l.mu.Lock()
	l.lastError = err
	handler := l.onReloadError
	l.mu.Unlock()
	if handler != nil {
		func() {
			defer func() { _ = recover() }()
			handler(err)
		}()
	}
}

func (l *Loader) safeOnChange(handler ChangeHandler, cfg *Config) {
	defer func() { _ = recover() }()
	handler(cfg)
}

func fileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func configsEqual(a, b *Config) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Version != b.Version ||
		a.BotToken != b.BotToken ||
		a.SQLitePath != b.SQLitePath ||
		a.LMDBIDPath != b.LMDBIDPath ||
		a.LMDBPhonePath != b.LMDBPhonePath ||
		a.LMDBUsernamePath != b.LMDBUsernamePath ||
		a.LogLevel != b.LogLevel ||
		a.WorkerCount != b.WorkerCount ||
		a.BatchSize != b.BatchSize ||
		a.SearchTimeout != b.SearchTimeout ||
		a.PointsPerSearch != b.PointsPerSearch ||
		a.MaxResults != b.MaxResults ||
		a.MaxSearchResult != b.MaxSearchResult ||
		a.CacheTTL != b.CacheTTL ||
		a.RateLimit != b.RateLimit {
		return false
	}
	if len(a.BotOwnerIDs) != len(b.BotOwnerIDs) {
		return false
	}
	for i := range a.BotOwnerIDs {
		if a.BotOwnerIDs[i] != b.BotOwnerIDs[i] {
			return false
		}
	}
	return true
}
