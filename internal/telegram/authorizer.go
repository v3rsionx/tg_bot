package telegram

import (
	"context"
)

// StaticAuthorizer is a simple DI-friendly Authorizer implementation.
type StaticAuthorizer struct {
	owners   map[int64]struct{}
	allowAll bool
}

// NewStaticAuthorizer constructs an authorizer from owner IDs.
// When allowAll is true, Authorize accepts every user.
func NewStaticAuthorizer(ownerIDs []int64, allowAll bool) *StaticAuthorizer {
	owners := make(map[int64]struct{}, len(ownerIDs))
	for _, id := range ownerIDs {
		if id > 0 {
			owners[id] = struct{}{}
		}
	}
	return &StaticAuthorizer{owners: owners, allowAll: allowAll}
}

// IsOwner reports whether userID is configured as an owner.
func (a *StaticAuthorizer) IsOwner(ctx context.Context, userID int64) bool {
	_ = ctx
	if a == nil {
		return false
	}
	_, ok := a.owners[userID]
	return ok
}

// Authorize validates ordinary bot access for userID.
func (a *StaticAuthorizer) Authorize(ctx context.Context, userID int64) error {
	_ = ctx
	if a == nil {
		return ErrUnauthorized
	}
	if a.allowAll || len(a.owners) == 0 {
		return nil
	}
	if _, ok := a.owners[userID]; ok {
		return nil
	}
	return ErrUnauthorized
}

var _ Authorizer = (*StaticAuthorizer)(nil)
