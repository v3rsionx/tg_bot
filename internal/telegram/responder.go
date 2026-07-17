package telegram

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/v3rsionx/tg_bot/internal/telegram/handlers"
)

// BotResponder adapts *bot.Bot to handlers.Responder.
type BotResponder struct {
	client *bot.Bot
}

// NewBotResponder constructs a BotResponder.
func NewBotResponder(client *bot.Bot) *BotResponder {
	return &BotResponder{client: client}
}

// ReplyText sends a plain-text message.
func (r *BotResponder) ReplyText(ctx context.Context, chatID int64, text string) error {
	if r == nil || r.client == nil {
		return ErrNilBot
	}
	_, err := r.client.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})
	return err
}

// AnswerCallback acknowledges a callback query.
func (r *BotResponder) AnswerCallback(ctx context.Context, callbackID, text string) error {
	if r == nil || r.client == nil {
		return ErrNilBot
	}
	_, err := r.client.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
		Text:            text,
	})
	return err
}

var _ handlers.Responder = (*BotResponder)(nil)
