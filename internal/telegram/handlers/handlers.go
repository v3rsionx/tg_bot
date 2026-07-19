package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Handlers is the command/message/callback registry implementation.
type Handlers struct {
	deps Dependencies
}

// New constructs Handlers with dependency injection.
func New(deps Dependencies) *Handlers {
	return &Handlers{deps: deps.withDefaults()}
}

// SetResponder injects the outbound responder after bot construction.
func (h *Handlers) SetResponder(responder Responder) {
	h.deps.Responder = responder
}

// Start handles /start.
func (h *Handlers) Start() bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID, userID, ok := messageMeta(update)
		if !ok {
			return
		}
		_ = userID
		h.reply(ctx, chatID, "Welcome. Use /help to see available commands.")
	}
}

// Help handles /help.
func (h *Handlers) Help() bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID, _, ok := messageMeta(update)
		if !ok {
			return
		}
		h.reply(ctx, chatID, strings.Join([]string{
			"Available commands:",
			"/start - start the bot",
			"/help - show help",
			"/profile - show profile",
			"/history - show recent history",
			"/admin - owner admin panel",
		}, "\n"))
	}
}

// Profile handles /profile through the Points interface.
func (h *Handlers) Profile() bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID, userID, ok := messageMeta(update)
		if !ok {
			return
		}
		if h.deps.Points == nil {
			h.reply(ctx, chatID, "Profile service is not configured.")
			return
		}
		balance, err := h.deps.Points.Balance(ctx, userID)
		if err != nil {
			h.deps.Logger.Errorf("profile points: %v", err)
			h.reply(ctx, chatID, "Unable to load profile.")
			return
		}
		h.reply(ctx, chatID, fmt.Sprintf("Profile\nUser ID: %d\nPoints: %d", userID, balance))
	}
}

// History handles /history through the History interface.
func (h *Handlers) History() bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID, userID, ok := messageMeta(update)
		if !ok {
			return
		}
		if h.deps.History == nil {
			h.reply(ctx, chatID, "History service is not configured.")
			return
		}
		items, err := h.deps.History.Recent(ctx, userID, h.deps.HistoryLimit)
		if err != nil {
			h.deps.Logger.Errorf("history: %v", err)
			h.reply(ctx, chatID, "Unable to load history.")
			return
		}
		if len(items) == 0 {
			h.reply(ctx, chatID, "No history yet.")
			return
		}
		var bld strings.Builder
		bld.WriteString("Recent history:\n")
		for i, item := range items {
			bld.WriteString(fmt.Sprintf("%d. [%s] %s (%s)\n", i+1, item.QueryType, item.Query, item.CreatedAt))
		}
		h.reply(ctx, chatID, strings.TrimSpace(bld.String()))
	}
}

// Admin handles /admin for owners only.
func (h *Handlers) Admin() bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID, userID, ok := messageMeta(update)
		if !ok {
			return
		}
		if h.deps.Authorizer == nil || !h.deps.Authorizer.IsOwner(ctx, userID) {
			h.reply(ctx, chatID, "Forbidden.")
			return
		}
		h.reply(ctx, chatID, "Admin panel ready.")
	}
}

// UnknownCommand handles unrecognized slash commands.
func (h *Handlers) UnknownCommand() bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID, _, ok := messageMeta(update)
		if !ok {
			return
		}
		h.reply(ctx, chatID, "Unknown command. Use /help.")
	}
}

// Message handles non-command text through the Search interface.
func (h *Handlers) Message() bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		chatID, userID, ok := messageMeta(update)
		if !ok || update.Message == nil {
			return
		}
		query := strings.TrimSpace(update.Message.Text)
		if query == "" {
			return
		}
		if h.deps.Search == nil {
			h.reply(ctx, chatID, "Search service is not configured.")
			return
		}
		result, err := h.deps.Search.ExactLookup(ctx, userID, query)
		if err != nil {
			h.deps.Logger.Errorf("search: %v", err)
			h.reply(ctx, chatID, "Search failed.")
			return
		}
		if !result.Found {
			h.reply(ctx, chatID, "No exact match found.")
			return
		}
		h.reply(ctx, chatID, formatSearchResult(result))
	}
}

// Callback handles callback queries.
func (h *Handlers) Callback() bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update == nil || update.CallbackQuery == nil {
			return
		}
		callbackID := update.CallbackQuery.ID
		data := update.CallbackQuery.Data
		h.deps.Logger.Infof("callback data=%s", data)
		if h.deps.Responder != nil {
			_ = h.deps.Responder.AnswerCallback(ctx, callbackID, "OK")
		}
		if update.CallbackQuery.Message.Message != nil {
			h.reply(ctx, update.CallbackQuery.Message.Message.Chat.ID, "Callback received.")
		}
	}
}

// Default routes unmatched updates to unknown-command or text handlers.
func (h *Handlers) Default() bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update == nil || update.Message == nil {
			if update != nil && update.CallbackQuery != nil {
				h.Callback()(ctx, b, update)
			}
			return
		}
		parsed := ParseCommand(update.Message.Text)
		if parsed.IsCommand {
			h.UnknownCommand()(ctx, b, update)
			return
		}
		h.Message()(ctx, b, update)
	}
}

// ErrorHandler logs transport-level bot errors.
func (h *Handlers) ErrorHandler() bot.ErrorsHandler {
	return func(err error) {
		if err == nil {
			return
		}
		h.deps.Logger.Errorf("bot error: %v", err)
	}
}

// formatSearchResult renders an exact-lookup hit for Telegram replies.
func formatSearchResult(result SearchResult) string {
	extras := result.Extras
	if extras == "{}" {
		extras = ""
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("ID: %s\n", result.ID))
	b.WriteString(fmt.Sprintf("Name: %s\n", result.Name))
	b.WriteString(fmt.Sprintf("Phone: %s\n", result.Phone))
	b.WriteString(fmt.Sprintf("Username: %s\n", result.Username))
	b.WriteString(fmt.Sprintf("Extras: %s", extras))
	return b.String()
}

// reply sends text using the injected responder when available.
func (h *Handlers) reply(ctx context.Context, chatID int64, text string) {
	if h.deps.Responder == nil {
		h.deps.Logger.Warnf("responder is nil; drop message chat_id=%d", chatID)
		return
	}
	if err := h.deps.Responder.ReplyText(ctx, chatID, text); err != nil {
		h.deps.Logger.Errorf("reply chat_id=%d: %v", chatID, err)
	}
}

// messageMeta extracts chat and user identifiers from a message update.
func messageMeta(update *models.Update) (chatID, userID int64, ok bool) {
	if update == nil || update.Message == nil {
		return 0, 0, false
	}
	chatID = update.Message.Chat.ID
	if update.Message.From != nil {
		userID = update.Message.From.ID
	}
	return chatID, userID, true
}
