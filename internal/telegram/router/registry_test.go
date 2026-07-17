package router_test

import (
	"context"
	"testing"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/v3rsionx/tg_bot/internal/telegram/router"
)

// TestRegistryRegisterAndGet verifies command registry behavior.
func TestRegistryRegisterAndGet(t *testing.T) {
	registry := router.NewRegistry()
	registry.Register("start", func(ctx context.Context, b *bot.Bot, update *models.Update) {})
	if _, ok := registry.Get("start"); !ok {
		t.Fatal("expected start handler")
	}
	names := registry.Commands()
	if len(names) != 1 || names[0] != "start" {
		t.Fatalf("Commands() = %#v", names)
	}
}
