package middleware

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Authorization returns middleware that enforces the Authorizer port.
// When authorizer is nil, all users are allowed.
func Authorization(authorizer Authorizer) bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if authorizer == nil {
				next(ctx, b, update)
				return
			}
			userID := userIDFromUpdate(update)
			if userID == 0 {
				next(ctx, b, update)
				return
			}
			if err := authorizer.Authorize(ctx, userID); err != nil {
				return
			}
			next(ctx, b, update)
		}
	}
}

// userIDFromUpdate extracts the acting Telegram user ID.
func userIDFromUpdate(update *models.Update) int64 {
	if update == nil {
		return 0
	}
	if update.Message != nil && update.Message.From != nil {
		return update.Message.From.ID
	}
	if update.CallbackQuery != nil {
		return update.CallbackQuery.From.ID
	}
	return 0
}
