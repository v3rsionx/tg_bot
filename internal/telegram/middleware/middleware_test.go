package middleware_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/v3rsionx/tg_bot/internal/telegram/middleware"
)

type stubLogger struct {
	infos int
}

func (s *stubLogger) Debugf(format string, args ...any) {}
func (s *stubLogger) Infof(format string, args ...any)  { s.infos++ }
func (s *stubLogger) Warnf(format string, args ...any)  {}
func (s *stubLogger) Errorf(format string, args ...any) {}

type stubAuthorizer struct {
	err error
}

func (s stubAuthorizer) Authorize(ctx context.Context, userID int64) error {
	return s.err
}

type stubLimiter struct {
	err error
}

func (s stubLimiter) Allow(ctx context.Context, userID int64) error {
	return s.err
}

// TestLoggingMiddlewareInvokesNext ensures logging middleware continues the chain.
func TestLoggingMiddlewareInvokesNext(t *testing.T) {
	log := &stubLogger{}
	called := false
	handler := middleware.Logging(log)(func(ctx context.Context, b *bot.Bot, update *models.Update) {
		called = true
	})
	handler(context.Background(), nil, &models.Update{
		Message: &models.Message{
			From: &models.User{ID: 7},
			Chat: models.Chat{ID: 9},
			Text: "hi",
		},
	})
	if !called {
		t.Fatal("next handler was not called")
	}
	if log.infos == 0 {
		t.Fatal("expected log output")
	}
}

// TestAuthorizationMiddlewareBlocksUnauthorizedUsers verifies auth short-circuit.
func TestAuthorizationMiddlewareBlocksUnauthorizedUsers(t *testing.T) {
	called := false
	handler := middleware.Authorization(stubAuthorizer{err: errors.New("denied")})(
		func(ctx context.Context, b *bot.Bot, update *models.Update) { called = true },
	)
	handler(context.Background(), nil, &models.Update{
		Message: &models.Message{From: &models.User{ID: 1}, Chat: models.Chat{ID: 1}},
	})
	if called {
		t.Fatal("unauthorized update should not reach next handler")
	}
}

// TestRateLimitMiddlewareBlocksLimitedUsers verifies rate-limit short-circuit.
func TestRateLimitMiddlewareBlocksLimitedUsers(t *testing.T) {
	called := false
	handler := middleware.RateLimit(stubLimiter{err: errors.New("limited")})(
		func(ctx context.Context, b *bot.Bot, update *models.Update) { called = true },
	)
	handler(context.Background(), nil, &models.Update{
		Message: &models.Message{From: &models.User{ID: 1}, Chat: models.Chat{ID: 1}},
	})
	if called {
		t.Fatal("rate-limited update should not reach next handler")
	}
}

// TestRecoveryMiddlewareCatchesPanics verifies panic isolation.
func TestRecoveryMiddlewareCatchesPanics(t *testing.T) {
	handler := middleware.Recovery(&stubLogger{})(func(ctx context.Context, b *bot.Bot, update *models.Update) {
		panic("boom")
	})
	handler(context.Background(), nil, &models.Update{})
}
