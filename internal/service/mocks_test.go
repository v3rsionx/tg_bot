package service_test

import (
	"context"
	"sync"
	"time"

	"github.com/v3rsionx/tg_bot/internal/models"
	"github.com/v3rsionx/tg_bot/internal/repository"
	"github.com/v3rsionx/tg_bot/internal/search"
	"github.com/v3rsionx/tg_bot/internal/service"
)

type memoryUsers struct {
	mu   sync.Mutex
	data map[int64]*models.User
}

func newMemoryUsers() *memoryUsers {
	return &memoryUsers{data: make(map[int64]*models.User)}
}

func (m *memoryUsers) Create(ctx context.Context, user *models.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[user.ID]; ok {
		return repository.ErrConflict
	}
	clone := *user
	m.data[user.ID] = &clone
	return nil
}

func (m *memoryUsers) Upsert(ctx context.Context, user *models.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	clone := *user
	m.data[user.ID] = &clone
	return nil
}

func (m *memoryUsers) GetByID(ctx context.Context, id int64) (*models.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, ok := m.data[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	clone := *user
	return &clone, nil
}

func (m *memoryUsers) Update(ctx context.Context, user *models.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[user.ID]; !ok {
		return repository.ErrNotFound
	}
	clone := *user
	m.data[user.ID] = &clone
	return nil
}

func (m *memoryUsers) UpdatePoints(ctx context.Context, userID int64, points int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, ok := m.data[userID]
	if !ok {
		return repository.ErrNotFound
	}
	user.Points = points
	user.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *memoryUsers) SetBanned(ctx context.Context, userID int64, banned bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, ok := m.data[userID]
	if !ok {
		return repository.ErrNotFound
	}
	user.IsBanned = banned
	return nil
}

func (m *memoryUsers) Exists(ctx context.Context, id int64) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.data[id]
	return ok, nil
}

type memoryTx struct {
	mu   sync.Mutex
	data []models.Transaction
}

func (m *memoryTx) Create(ctx context.Context, tx *models.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	tx.ID = int64(len(m.data) + 1)
	m.data = append(m.data, *tx)
	return nil
}

func (m *memoryTx) GetByID(ctx context.Context, id int64) (*models.Transaction, error) {
	return nil, repository.ErrNotFound
}

func (m *memoryTx) ListByUserID(ctx context.Context, userID int64, limit, offset int) ([]models.Transaction, error) {
	return nil, nil
}

type memoryHistory struct {
	mu   sync.Mutex
	data []models.SearchHistory
}

func (m *memoryHistory) Create(ctx context.Context, entry *models.SearchHistory) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry.ID = int64(len(m.data) + 1)
	m.data = append(m.data, *entry)
	return nil
}

func (m *memoryHistory) GetByID(ctx context.Context, id int64) (*models.SearchHistory, error) {
	return nil, repository.ErrNotFound
}

func (m *memoryHistory) ListByUserID(ctx context.Context, userID int64, limit, offset int) ([]models.SearchHistory, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]models.SearchHistory, 0)
	for i := len(m.data) - 1; i >= 0; i-- {
		if m.data[i].UserID != userID {
			continue
		}
		out = append(out, m.data[i])
	}
	if offset > len(out) {
		return nil, nil
	}
	out = out[offset:]
	if limit < len(out) {
		out = out[:limit]
	}
	return out, nil
}

type stubEngine struct {
	byID map[string]search.Result
}

func (e stubEngine) SearchByID(ctx context.Context, id string) (search.Result, error) {
	if result, ok := e.byID[id]; ok {
		return result, nil
	}
	return search.Result{}, search.ErrNotFound
}

func (e stubEngine) SearchByPhone(ctx context.Context, phone string) (search.Result, error) {
	return search.Result{}, search.ErrNotFound
}

func (e stubEngine) SearchByUsername(ctx context.Context, username string) (search.Result, error) {
	return search.Result{}, search.ErrNotFound
}

func (e stubEngine) Stats() search.Statistics { return search.Statistics{} }

type stubSender struct {
	mu      sync.Mutex
	messages []string
}

func (s *stubSender) SendText(ctx context.Context, chatID int64, text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, text)
	return nil
}

var (
	_ service.UserRepository        = (*memoryUsers)(nil)
	_ service.TransactionRepository = (*memoryTx)(nil)
	_ service.HistoryRepository     = (*memoryHistory)(nil)
	_ service.SearchEngine          = stubEngine{}
	_ service.MessageSender         = (*stubSender)(nil)
)
