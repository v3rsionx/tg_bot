package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/v3rsi/tgbot-versionx/internal/models"
	"github.com/v3rsi/tgbot-versionx/internal/repository"
)

// UserService manages bot user registration and profiles.
type UserService struct {
	users  UserRepository
	log    Logger
	stats  *statsAccumulator
	known  sync.Map
}

// NewUserService constructs a UserService with dependency injection.
func NewUserService(users UserRepository, log Logger, stats *statsAccumulator) *UserService {
	if log == nil {
		log = NopLogger{}
	}
	if stats == nil {
		stats = newStatsAccumulator()
	}
	return &UserService{users: users, log: log, stats: stats}
}

// Register upserts a Telegram user profile.
func (s *UserService) Register(ctx context.Context, user *models.User) error {
	if user == nil || user.ID <= 0 {
		return fmt.Errorf("%w: user", ErrInvalidInput)
	}
	exists, err := s.users.Exists(ctx, user.ID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now
	if err := s.users.Upsert(ctx, user); err != nil {
		return err
	}
	if !exists {
		if _, loaded := s.known.LoadOrStore(user.ID, struct{}{}); !loaded {
			s.stats.addRegistered(1)
		}
		s.log.Infof("user registered id=%d", user.ID)
	}
	return nil
}

// Profile returns the user profile, creating a default user when missing.
func (s *UserService) Profile(ctx context.Context, userID int64) (*models.User, error) {
	if err := s.CreateUserIfNotExists(ctx, userID); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID)
}

// Exists reports whether the user is present.
func (s *UserService) Exists(ctx context.Context, userID int64) (bool, error) {
	if userID <= 0 {
		return false, fmt.Errorf("%w: user id", ErrInvalidInput)
	}
	return s.users.Exists(ctx, userID)
}

// Get returns a user by ID.
func (s *UserService) Get(ctx context.Context, userID int64) (*models.User, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("%w: user id", ErrInvalidInput)
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

// Update persists mutable user fields.
func (s *UserService) Update(ctx context.Context, user *models.User) error {
	if user == nil || user.ID <= 0 {
		return fmt.Errorf("%w: user", ErrInvalidInput)
	}
	user.UpdatedAt = time.Now().UTC()
	if err := s.users.Update(ctx, user); err != nil {
		if err == repository.ErrNotFound {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// CreateUserIfNotExists ensures a user row exists.
func (s *UserService) CreateUserIfNotExists(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("%w: user id", ErrInvalidInput)
	}
	exists, err := s.users.Exists(ctx, userID)
	if err != nil {
		return err
	}
	if exists {
		s.known.LoadOrStore(userID, struct{}{})
		return nil
	}
	now := time.Now().UTC()
	user := &models.User{
		ID:        userID,
		Points:    0,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.users.Create(ctx, user); err != nil {
		// Concurrent create may race; treat existing row as success.
		if existsNow, existsErr := s.users.Exists(ctx, userID); existsErr == nil && existsNow {
			return nil
		}
		return err
	}
	if _, loaded := s.known.LoadOrStore(userID, struct{}{}); !loaded {
		s.stats.addRegistered(1)
	}
	s.log.Infof("user auto-created id=%d", userID)
	return nil
}

// KnownCount returns the number of users observed by this process.
func (s *UserService) KnownCount() int64 {
	var n int64
	s.known.Range(func(key, value any) bool {
		n++
		return true
	})
	return n
}

// ListKnownIDs returns user IDs observed by this process.
func (s *UserService) ListKnownIDs() []int64 {
	ids := make([]int64, 0)
	s.known.Range(func(key, value any) bool {
		if id, ok := key.(int64); ok {
			ids = append(ids, id)
		}
		return true
	})
	return ids
}
