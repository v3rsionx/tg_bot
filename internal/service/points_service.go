package service

import (
	"context"
	"fmt"
	"time"

	"github.com/v3rsi/tgbot-versionx/internal/models"
	"github.com/v3rsi/tgbot-versionx/internal/repository"
)

// PointsService manages user point balances and ledger entries.
type PointsService struct {
	users        UserRepository
	transactions TransactionRepository
	usersSvc     *UserService
	log          Logger
}

// NewPointsService constructs a PointsService with dependency injection.
func NewPointsService(
	users UserRepository,
	transactions TransactionRepository,
	usersSvc *UserService,
	log Logger,
) *PointsService {
	if log == nil {
		log = NopLogger{}
	}
	return &PointsService{
		users:        users,
		transactions: transactions,
		usersSvc:     usersSvc,
		log:          log,
	}
}

// CreateUserIfNotExists ensures a user row exists before point operations.
func (s *PointsService) CreateUserIfNotExists(ctx context.Context, userID int64) error {
	if s.usersSvc != nil {
		return s.usersSvc.CreateUserIfNotExists(ctx, userID)
	}
	exists, err := s.users.Exists(ctx, userID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	now := time.Now().UTC()
	return s.users.Create(ctx, &models.User{ID: userID, CreatedAt: now, UpdatedAt: now})
}

// GetBalance returns the current points balance.
func (s *PointsService) GetBalance(ctx context.Context, userID int64) (int64, error) {
	if err := s.CreateUserIfNotExists(ctx, userID); err != nil {
		return 0, err
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if err == repository.ErrNotFound {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return user.Points, nil
}

// HasEnoughPoints reports whether the user can afford amount.
func (s *PointsService) HasEnoughPoints(ctx context.Context, userID, amount int64) (bool, error) {
	if amount < 0 {
		return false, fmt.Errorf("%w: amount", ErrInvalidInput)
	}
	balance, err := s.GetBalance(ctx, userID)
	if err != nil {
		return false, err
	}
	return balance >= amount, nil
}

// AddPoints credits points to a user and writes a ledger entry.
func (s *PointsService) AddPoints(ctx context.Context, userID, amount int64, reason string) error {
	if amount <= 0 {
		return fmt.Errorf("%w: amount", ErrInvalidInput)
	}
	if err := s.CreateUserIfNotExists(ctx, userID); err != nil {
		return err
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	newBalance := user.Points + amount
	if err := s.users.UpdatePoints(ctx, userID, newBalance); err != nil {
		return err
	}
	if s.transactions != nil {
		_ = s.transactions.Create(ctx, &models.Transaction{
			UserID:    userID,
			Amount:    amount,
			Type:      models.TransactionTypeCredit,
			Reason:    reason,
			CreatedAt: time.Now().UTC(),
		})
	}
	s.log.Infof("points credited user_id=%d amount=%d reason=%q balance=%d", userID, amount, reason, newBalance)
	return nil
}

// RemovePoints debits points from a user and writes a ledger entry.
func (s *PointsService) RemovePoints(ctx context.Context, userID, amount int64, reason string) error {
	if amount <= 0 {
		return fmt.Errorf("%w: amount", ErrInvalidInput)
	}
	if err := s.CreateUserIfNotExists(ctx, userID); err != nil {
		return err
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.Points < amount {
		return ErrInsufficientPoints
	}
	newBalance := user.Points - amount
	if err := s.users.UpdatePoints(ctx, userID, newBalance); err != nil {
		return err
	}
	if s.transactions != nil {
		_ = s.transactions.Create(ctx, &models.Transaction{
			UserID:    userID,
			Amount:    amount,
			Type:      models.TransactionTypeDebit,
			Reason:    reason,
			CreatedAt: time.Now().UTC(),
		})
	}
	s.log.Infof("points debited user_id=%d amount=%d reason=%q balance=%d", userID, amount, reason, newBalance)
	return nil
}

// Balance is an alias for GetBalance used by transport adapters.
func (s *PointsService) Balance(ctx context.Context, userID int64) (int64, error) {
	return s.GetBalance(ctx, userID)
}
