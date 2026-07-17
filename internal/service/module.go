package service

import (
	"context"
	"fmt"
)

// Module wires all business services with shared dependencies.
type Module struct {
	Config   Config
	Stats    *statsAccumulator
	Users    *UserService
	Points   *PointsService
	History  *HistoryService
	Search   *SearchService
	Admin    *AdminService
	Access   *AccessService
	RateLimit *SearchRateLimiter
}

// ModuleDeps contains injectable infrastructure ports for Module construction.
type ModuleDeps struct {
	Users        UserRepository
	Transactions TransactionRepository
	History      HistoryRepository
	Engine       SearchEngine
	Sender       MessageSender
	Directory    UserDirectory
	Logger       Logger
}

// NewModule constructs the business module graph.
func NewModule(cfg Config, deps ModuleDeps) (*Module, error) {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if deps.Users == nil {
		return nil, fmt.Errorf("service: Users repository is required")
	}
	if deps.History == nil {
		return nil, fmt.Errorf("service: History repository is required")
	}
	if deps.Engine == nil {
		return nil, fmt.Errorf("service: SearchEngine is required")
	}
	if deps.Logger == nil {
		deps.Logger = NopLogger{}
	}

	stats := newStatsAccumulator()
	users := NewUserService(deps.Users, deps.Logger, stats)
	points := NewPointsService(deps.Users, deps.Transactions, users, deps.Logger)
	history := NewHistoryService(deps.History, deps.Logger)
	searchSvc, err := NewSearchService(cfg, deps.Engine, users, points, history, stats, deps.Logger)
	if err != nil {
		return nil, err
	}

	directory := deps.Directory
	if directory == nil {
		directory = userDirectory{users: users}
	}

	admin := NewAdminService(cfg, users, points, history, stats, deps.Engine, deps.Sender, directory, deps.Logger)
	access := NewAccessService(cfg, deps.Users, deps.Logger)
	limiter := NewSearchRateLimiter(cfg)

	return &Module{
		Config:    cfg,
		Stats:     stats,
		Users:     users,
		Points:    points,
		History:   history,
		Search:    searchSvc,
		Admin:     admin,
		Access:    access,
		RateLimit: limiter,
	}, nil
}

// userDirectory adapts UserService to UserDirectory.
type userDirectory struct {
	users *UserService
}

// ListUserIDs returns known user IDs from the user service.
func (d userDirectory) ListUserIDs(ctx context.Context) ([]int64, error) {
	_ = ctx
	if d.users == nil {
		return nil, nil
	}
	return d.users.ListKnownIDs(), nil
}
