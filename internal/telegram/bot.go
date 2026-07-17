package telegram

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/v3rsi/tgbot-versionx/internal/telegram/handlers"
	"github.com/v3rsi/tgbot-versionx/internal/telegram/middleware"
	"github.com/v3rsi/tgbot-versionx/internal/telegram/router"
)

// Bot is the injectable Telegram transport facade.
type Bot struct {
	cfg    Config
	deps   Dependencies
	client *bot.Bot

	mu      sync.Mutex
	started bool
	closed  bool
}

// New constructs and initializes a Telegram bot with DI-wired collaborators.
func New(cfg Config, deps Dependencies) (*Bot, error) {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if err := deps.Validate(); err != nil {
		return nil, err
	}

	var searchPort handlers.Search
	if deps.Search != nil {
		searchPort = searchBridge{inner: deps.Search}
	}
	var historyPort handlers.History
	if deps.History != nil {
		historyPort = historyBridge{inner: deps.History}
	}

	handlerSet := handlers.New(handlers.Dependencies{
		Logger:       deps.Logger,
		Authorizer:   deps.Authorizer,
		Points:       deps.Points,
		Search:       searchPort,
		History:      historyPort,
		HistoryLimit: cfg.HistoryLimit,
	})

	middlewares := []bot.Middleware{
		middleware.Recovery(deps.Logger),
		middleware.Logging(deps.Logger),
		middleware.Authorization(deps.Authorizer),
		middleware.RateLimit(deps.RateLimiter),
	}

	opts := []bot.Option{
		bot.WithMiddlewares(middlewares...),
		bot.WithErrorsHandler(handlerSet.ErrorHandler()),
		bot.WithDefaultHandler(handlerSet.Default()),
	}

	client, err := bot.New(cfg.Token, opts...)
	if err != nil {
		return nil, fmt.Errorf("telegram: create bot: %w", err)
	}
	handlerSet.SetResponder(NewBotResponder(client))

	registry := router.NewRegistry()
	registry.Register("start", handlerSet.Start())
	registry.Register("help", handlerSet.Help())
	registry.Register("profile", handlerSet.Profile())
	registry.Register("history", handlerSet.History())
	registry.Register("admin", handlerSet.Admin())

	if err := router.Apply(client, registry, handlerSet); err != nil {
		return nil, err
	}

	return &Bot{
		cfg:    cfg,
		deps:   deps,
		client: client,
	}, nil
}

// Client returns the underlying go-telegram bot client.
func (b *Bot) Client() *bot.Bot {
	if b == nil {
		return nil
	}
	return b.client
}

// Start begins long-polling and blocks until ctx is cancelled.
func (b *Bot) Start(ctx context.Context) error {
	if b == nil || b.client == nil {
		return ErrNilBot
	}

	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return ErrClosed
	}
	if b.started {
		b.mu.Unlock()
		return fmt.Errorf("telegram: bot already started")
	}
	b.started = true
	b.mu.Unlock()

	b.deps.Logger.Infof("telegram bot starting")
	b.client.Start(ctx)
	b.deps.Logger.Infof("telegram bot stopped")
	return nil
}

// Shutdown marks the bot closed for graceful teardown.
// Cancel the context passed to Start to stop long-polling.
func (b *Bot) Shutdown(ctx context.Context) error {
	if b == nil {
		return ErrNilBot
	}
	_ = ctx

	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	b.deps.Logger.Infof("telegram bot shutdown requested")
	return nil
}
