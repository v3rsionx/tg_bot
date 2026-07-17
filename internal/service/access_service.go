package service

import (
	"context"

	"github.com/v3rsi/tgbot-versionx/internal/repository"
)

// AccessService authorizes bot access and owner checks.
type AccessService struct {
	owners map[int64]struct{}
	users  UserRepository
	log    Logger
}

// NewAccessService constructs an AccessService.
func NewAccessService(cfg Config, users UserRepository, log Logger) *AccessService {
	if log == nil {
		log = NopLogger{}
	}
	return &AccessService{
		owners: ownerSet(cfg.OwnerIDs),
		users:  users,
		log:    log,
	}
}

// IsOwner reports whether userID is a configured owner.
func (s *AccessService) IsOwner(ctx context.Context, userID int64) bool {
	_ = ctx
	_, ok := s.owners[userID]
	return ok
}

// Authorize validates ordinary bot access and bans.
func (s *AccessService) Authorize(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return ErrUnauthorized
	}
	if s.IsOwner(ctx, userID) {
		return nil
	}
	if s.users == nil {
		return nil
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil
		}
		s.log.Errorf("authorize lookup user=%d: %v", userID, err)
		return err
	}
	if user.IsBanned {
		return ErrBanned
	}
	return nil
}
