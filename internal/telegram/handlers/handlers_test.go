package handlers_test

import (
	"context"
	"testing"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/v3rsi/tgbot-versionx/internal/telegram/handlers"
)

type stubResponder struct {
	texts []string
}

func (s *stubResponder) ReplyText(ctx context.Context, chatID int64, text string) error {
	s.texts = append(s.texts, text)
	return nil
}

func (s *stubResponder) AnswerCallback(ctx context.Context, callbackID, text string) error {
	return nil
}

type stubPoints struct{ balance int64 }

func (s stubPoints) Balance(ctx context.Context, userID int64) (int64, error) {
	return s.balance, nil
}

type stubSearch struct{ result handlers.SearchResult }

func (s stubSearch) ExactLookup(ctx context.Context, userID int64, query string) (handlers.SearchResult, error) {
	return s.result, nil
}

type stubAuthorizer struct{ owner bool }

func (s stubAuthorizer) IsOwner(ctx context.Context, userID int64) bool { return s.owner }

type stubLogger struct{}

func (stubLogger) Debugf(format string, args ...any) {}
func (stubLogger) Infof(format string, args ...any)  {}
func (stubLogger) Warnf(format string, args ...any)  {}
func (stubLogger) Errorf(format string, args ...any) {}

func messageUpdate(userID int64, text string) *models.Update {
	return &models.Update{
		Message: &models.Message{
			From: &models.User{ID: userID},
			Chat: models.Chat{ID: 100},
			Text: text,
		},
	}
}

// TestStartAndHelpHandlersReply verifies basic command replies.
func TestStartAndHelpHandlersReply(t *testing.T) {
	responder := &stubResponder{}
	h := handlers.New(handlers.Dependencies{Logger: stubLogger{}, Responder: responder})

	h.Start()(context.Background(), nil, messageUpdate(1, "/start"))
	h.Help()(context.Background(), nil, messageUpdate(1, "/help"))
	if len(responder.texts) != 2 {
		t.Fatalf("replies = %d, want 2", len(responder.texts))
	}
}

// TestProfileHandlerUsesPointsInterface verifies points DI.
func TestProfileHandlerUsesPointsInterface(t *testing.T) {
	responder := &stubResponder{}
	h := handlers.New(handlers.Dependencies{
		Logger:    stubLogger{},
		Responder: responder,
		Points:    stubPoints{balance: 42},
	})
	h.Profile()(context.Background(), nil, messageUpdate(7, "/profile"))
	if len(responder.texts) != 1 || responder.texts[0] == "" {
		t.Fatalf("unexpected replies: %#v", responder.texts)
	}
}

// TestAdminHandlerEnforcesOwnerCheck verifies authorization interface usage.
func TestAdminHandlerEnforcesOwnerCheck(t *testing.T) {
	responder := &stubResponder{}
	h := handlers.New(handlers.Dependencies{
		Logger:     stubLogger{},
		Responder:  responder,
		Authorizer: stubAuthorizer{owner: false},
	})
	h.Admin()(context.Background(), nil, messageUpdate(7, "/admin"))
	if responder.texts[0] != "Forbidden." {
		t.Fatalf("reply = %q, want Forbidden.", responder.texts[0])
	}
}

// TestDefaultHandlerRoutesUnknownCommandAndText verifies fallback routing.
func TestDefaultHandlerRoutesUnknownCommandAndText(t *testing.T) {
	responder := &stubResponder{}
	h := handlers.New(handlers.Dependencies{
		Logger:    stubLogger{},
		Responder: responder,
		Search: stubSearch{result: handlers.SearchResult{
			Found: true, ID: "1", Phone: "+1", Username: "a",
		}},
	})

	h.Default()(context.Background(), (*bot.Bot)(nil), messageUpdate(1, "/unknown"))
	h.Default()(context.Background(), nil, messageUpdate(1, "+15551112222"))
	if len(responder.texts) != 2 {
		t.Fatalf("replies = %d, want 2", len(responder.texts))
	}
}
