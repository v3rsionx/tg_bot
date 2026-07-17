package middleware

import (
	"context"
	"runtime/debug"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Recovery returns middleware that converts panics into logged errors.
func Recovery(log Logger) bot.Middleware {
	if log == nil {
		log = nopLogger{}
	}
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			defer func() {
				if recovered := recover(); recovered != nil {
					log.Errorf("panic recovered: %v\n%s", recovered, debug.Stack())
				}
			}()
			next(ctx, b, update)
		}
	}
}
