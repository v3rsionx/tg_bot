package middleware

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Logging returns middleware that records basic update metadata.
func Logging(log Logger) bot.Middleware {
	if log == nil {
		log = nopLogger{}
	}
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			userID, chatID, kind := describeUpdate(update)
			log.Infof("update kind=%s user_id=%d chat_id=%d", kind, userID, chatID)
			next(ctx, b, update)
		}
	}
}

// describeUpdate extracts stable identifiers for logging.
func describeUpdate(update *models.Update) (userID, chatID int64, kind string) {
	if update == nil {
		return 0, 0, "nil"
	}
	switch {
	case update.Message != nil:
		kind = "message"
		if update.Message.From != nil {
			userID = update.Message.From.ID
		}
		chatID = update.Message.Chat.ID
	case update.CallbackQuery != nil:
		kind = "callback"
		userID = update.CallbackQuery.From.ID
		if update.CallbackQuery.Message.Message != nil {
			chatID = update.CallbackQuery.Message.Message.Chat.ID
		}
	default:
		kind = "other"
	}
	return userID, chatID, kind
}

type nopLogger struct{}

func (nopLogger) Debugf(format string, args ...any) {}
func (nopLogger) Infof(format string, args ...any)  {}
func (nopLogger) Warnf(format string, args ...any)  {}
func (nopLogger) Errorf(format string, args ...any) {}
