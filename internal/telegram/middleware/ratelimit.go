package middleware

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// RateLimit returns middleware that enforces the RateLimiter port.
// When limiter is nil, all users are allowed.
func RateLimit(limiter RateLimiter) bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if limiter == nil {
				next(ctx, b, update)
				return
			}
			userID := userIDFromUpdate(update)
			if userID == 0 {
				next(ctx, b, update)
				return
			}
			if err := limiter.Allow(ctx, userID); err != nil {
				return
			}
			next(ctx, b, update)
		}
	}
}
