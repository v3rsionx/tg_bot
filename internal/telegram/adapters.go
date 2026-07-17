package telegram

import (
	"context"

	"github.com/v3rsionx/tg_bot/internal/telegram/handlers"
)

// searchBridge adapts telegram.Search to handlers.Search.
type searchBridge struct {
	inner Search
}

// ExactLookup adapts the search port result type.
func (b searchBridge) ExactLookup(ctx context.Context, userID int64, query string) (handlers.SearchResult, error) {
	if b.inner == nil {
		return handlers.SearchResult{}, nil
	}
	result, err := b.inner.ExactLookup(ctx, userID, query)
	if err != nil {
		return handlers.SearchResult{}, err
	}
	return handlers.SearchResult{
		Found:    result.Found,
		ID:       result.ID,
		Phone:    result.Phone,
		Username: result.Username,
	}, nil
}

// historyBridge adapts telegram.History to handlers.History.
type historyBridge struct {
	inner History
}

// Recent adapts the history port result type.
func (b historyBridge) Recent(ctx context.Context, userID int64, limit int) ([]handlers.HistoryItem, error) {
	if b.inner == nil {
		return nil, nil
	}
	items, err := b.inner.Recent(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]handlers.HistoryItem, 0, len(items))
	for _, item := range items {
		out = append(out, handlers.HistoryItem{
			Query:     item.Query,
			QueryType: item.QueryType,
			CreatedAt: item.CreatedAt,
		})
	}
	return out, nil
}

var (
	_ handlers.Search  = searchBridge{}
	_ handlers.History = historyBridge{}
)
