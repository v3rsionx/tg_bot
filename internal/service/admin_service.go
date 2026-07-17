package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// AdminStats is the structured admin dashboard snapshot.
type AdminStats struct {
	App            AppStatistics
	EngineQueries  uint64
	EngineHits     uint64
	EngineMisses   uint64
	AverageLatency time.Duration
	GeneratedAt    time.Time
}

// BroadcastResult summarizes a broadcast attempt.
type BroadcastResult struct {
	Attempted int
	Sent      int
	Failed    int
}

// AdminService implements privileged owner operations.
type AdminService struct {
	cfg      Config
	owners   map[int64]struct{}
	users    *UserService
	points   *PointsService
	history  *HistoryService
	stats    *statsAccumulator
	engine   SearchEngine
	sender   MessageSender
	directory UserDirectory
	log      Logger
}

// NewAdminService constructs an AdminService with dependency injection.
func NewAdminService(
	cfg Config,
	users *UserService,
	points *PointsService,
	history *HistoryService,
	stats *statsAccumulator,
	engine SearchEngine,
	sender MessageSender,
	directory UserDirectory,
	log Logger,
) *AdminService {
	cfg = cfg.withDefaults()
	if log == nil {
		log = NopLogger{}
	}
	if stats == nil {
		stats = newStatsAccumulator()
	}
	return &AdminService{
		cfg:       cfg,
		owners:    ownerSet(cfg.OwnerIDs),
		users:     users,
		points:    points,
		history:   history,
		stats:     stats,
		engine:    engine,
		sender:    sender,
		directory: directory,
		log:       log,
	}
}

// ensureOwner rejects non-owner actors.
func (s *AdminService) ensureOwner(actorID int64) error {
	if _, ok := s.owners[actorID]; !ok {
		return ErrForbidden
	}
	return nil
}

// AddPoints credits points to a target user.
func (s *AdminService) AddPoints(ctx context.Context, actorID, userID, amount int64, reason string) error {
	if err := s.ensureOwner(actorID); err != nil {
		return err
	}
	if reason == "" {
		reason = "admin_add"
	}
	if err := s.points.AddPoints(ctx, userID, amount, reason); err != nil {
		return err
	}
	s.log.Infof("admin add points actor=%d user=%d amount=%d reason=%q", actorID, userID, amount, reason)
	return nil
}

// RemovePoints debits points from a target user.
func (s *AdminService) RemovePoints(ctx context.Context, actorID, userID, amount int64, reason string) error {
	if err := s.ensureOwner(actorID); err != nil {
		return err
	}
	if reason == "" {
		reason = "admin_remove"
	}
	if err := s.points.RemovePoints(ctx, userID, amount, reason); err != nil {
		return err
	}
	s.log.Infof("admin remove points actor=%d user=%d amount=%d reason=%q", actorID, userID, amount, reason)
	return nil
}

// Ban bans a target user.
func (s *AdminService) Ban(ctx context.Context, actorID, userID int64) error {
	if err := s.ensureOwner(actorID); err != nil {
		return err
	}
	if err := s.users.CreateUserIfNotExists(ctx, userID); err != nil {
		return err
	}
	user, err := s.users.Get(ctx, userID)
	if err != nil {
		return err
	}
	if user.IsBanned {
		return nil
	}
	user.IsBanned = true
	if err := s.users.Update(ctx, user); err != nil {
		return err
	}
	s.stats.addBanned(1)
	s.log.Infof("admin ban actor=%d user=%d", actorID, userID)
	return nil
}

// Unban removes a ban from a target user.
func (s *AdminService) Unban(ctx context.Context, actorID, userID int64) error {
	if err := s.ensureOwner(actorID); err != nil {
		return err
	}
	user, err := s.users.Get(ctx, userID)
	if err != nil {
		return err
	}
	if !user.IsBanned {
		return nil
	}
	user.IsBanned = false
	if err := s.users.Update(ctx, user); err != nil {
		return err
	}
	s.stats.addBanned(-1)
	s.log.Infof("admin unban actor=%d user=%d", actorID, userID)
	return nil
}

// Stats returns aggregated admin statistics.
func (s *AdminService) Stats(ctx context.Context, actorID int64) (AdminStats, error) {
	_ = ctx
	if err := s.ensureOwner(actorID); err != nil {
		return AdminStats{}, err
	}
	app := s.stats.snapshot()
	out := AdminStats{
		App:         app,
		GeneratedAt: time.Now().UTC(),
	}
	if s.engine != nil {
		engineStats := s.engine.Stats()
		out.EngineQueries = engineStats.Queries
		out.EngineHits = engineStats.Hits
		out.EngineMisses = engineStats.Misses
		out.AverageLatency = app.AverageLatency
		if out.AverageLatency == 0 {
			out.AverageLatency = engineStats.AverageLatency
		}
	}
	return out, nil
}

// UserCount returns the known user count.
func (s *AdminService) UserCount(ctx context.Context, actorID int64) (int64, error) {
	if err := s.ensureOwner(actorID); err != nil {
		return 0, err
	}
	if s.directory != nil {
		ids, err := s.directory.ListUserIDs(ctx)
		if err != nil {
			return 0, err
		}
		return int64(len(ids)), nil
	}
	if s.users != nil {
		return s.users.KnownCount(), nil
	}
	return s.stats.snapshot().RegisteredUsers, nil
}

// Broadcast sends a message to known users through MessageSender.
func (s *AdminService) Broadcast(ctx context.Context, actorID int64, message string) (BroadcastResult, error) {
	if err := s.ensureOwner(actorID); err != nil {
		return BroadcastResult{}, err
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return BroadcastResult{}, fmt.Errorf("%w: message", ErrInvalidInput)
	}
	if s.sender == nil {
		return BroadcastResult{}, fmt.Errorf("%w: message sender", ErrNotSupported)
	}

	var ids []int64
	var err error
	if s.directory != nil {
		ids, err = s.directory.ListUserIDs(ctx)
		if err != nil {
			return BroadcastResult{}, err
		}
	} else if s.users != nil {
		ids = s.users.ListKnownIDs()
	}
	result := BroadcastResult{Attempted: len(ids)}
	for _, id := range ids {
		if err := s.sender.SendText(ctx, id, message); err != nil {
			result.Failed++
			s.log.Errorf("broadcast send user=%d: %v", id, err)
			continue
		}
		result.Sent++
	}
	s.log.Infof("admin broadcast actor=%d attempted=%d sent=%d failed=%d", actorID, result.Attempted, result.Sent, result.Failed)
	return result, nil
}

// History returns recent searches for a target user.
func (s *AdminService) History(ctx context.Context, actorID, userID int64, limit int) ([]HistoryEntry, error) {
	if err := s.ensureOwner(actorID); err != nil {
		return nil, err
	}
	return s.history.LastSearches(ctx, userID, limit)
}

// Profile returns a target user profile.
func (s *AdminService) Profile(ctx context.Context, actorID, userID int64) (*UserProfile, error) {
	if err := s.ensureOwner(actorID); err != nil {
		return nil, err
	}
	user, err := s.users.Profile(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &UserProfile{
		ID:       user.ID,
		Username: user.Username,
		Points:   user.Points,
		IsBanned: user.IsBanned,
	}, nil
}

// UserProfile is a transport-friendly admin profile view.
type UserProfile struct {
	ID       int64
	Username string
	Points   int64
	IsBanned bool
}

// Execute parses and runs an admin command line for owners.
// Supported: /admin /stats /users /addpoint /removepoint /ban /unban /broadcast /history /profile
func (s *AdminService) Execute(ctx context.Context, actorID int64, command string, args []string) (string, error) {
	if err := s.ensureOwner(actorID); err != nil {
		return "Unauthorized.", ErrForbidden
	}
	command = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(command), "/"))
	switch command {
	case "admin":
		return "Admin panel ready.\nCommands: /stats /users /addpoint /removepoint /ban /unban /broadcast /history /profile", nil
	case "stats":
		stats, err := s.Stats(ctx, actorID)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(
			"Stats\nTotal: %d\nToday: %d\nSuccess: %d\nFailed: %d\nAvg latency: %s\nPoints used: %d",
			stats.App.TotalSearches,
			stats.App.TodaySearches,
			stats.App.SuccessfulSearches,
			stats.App.FailedSearches,
			stats.AverageLatency,
			stats.App.PointsSpent,
		), nil
	case "users":
		count, err := s.UserCount(ctx, actorID)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Users: %d", count), nil
	case "addpoint", "addpoints":
		userID, amount, err := parseUserAmount(args)
		if err != nil {
			return "", err
		}
		if err := s.AddPoints(ctx, actorID, userID, amount, "admin_addpoint"); err != nil {
			return "", err
		}
		return fmt.Sprintf("Added %d points to %d", amount, userID), nil
	case "removepoint", "removepoints":
		userID, amount, err := parseUserAmount(args)
		if err != nil {
			return "", err
		}
		if err := s.RemovePoints(ctx, actorID, userID, amount, "admin_removepoint"); err != nil {
			return "", err
		}
		return fmt.Sprintf("Removed %d points from %d", amount, userID), nil
	case "ban":
		userID, err := parseUserID(args)
		if err != nil {
			return "", err
		}
		if err := s.Ban(ctx, actorID, userID); err != nil {
			return "", err
		}
		return fmt.Sprintf("Banned %d", userID), nil
	case "unban":
		userID, err := parseUserID(args)
		if err != nil {
			return "", err
		}
		if err := s.Unban(ctx, actorID, userID); err != nil {
			return "", err
		}
		return fmt.Sprintf("Unbanned %d", userID), nil
	case "broadcast":
		if len(args) == 0 {
			return "", fmt.Errorf("%w: broadcast message", ErrInvalidInput)
		}
		result, err := s.Broadcast(ctx, actorID, strings.Join(args, " "))
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Broadcast sent=%d failed=%d", result.Sent, result.Failed), nil
	case "history":
		userID, err := parseUserID(args)
		if err != nil {
			return "", err
		}
		items, err := s.History(ctx, actorID, userID, s.cfg.DefaultHistoryLimit)
		if err != nil {
			return "", err
		}
		if len(items) == 0 {
			return "No history.", nil
		}
		var b strings.Builder
		b.WriteString("History:\n")
		for i, item := range items {
			b.WriteString(fmt.Sprintf("%d. [%s] %s found=%t\n", i+1, item.SearchType, item.Keyword, item.ResultFound))
		}
		return strings.TrimSpace(b.String()), nil
	case "profile":
		userID, err := parseUserID(args)
		if err != nil {
			return "", err
		}
		profile, err := s.Profile(ctx, actorID, userID)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Profile %d\nPoints: %d\nBanned: %t", profile.ID, profile.Points, profile.IsBanned), nil
	default:
		return "", fmt.Errorf("%w: unknown admin command", ErrInvalidInput)
	}
}

// parseUserID parses the first argument as a user ID.
func parseUserID(args []string) (int64, error) {
	if len(args) < 1 {
		return 0, fmt.Errorf("%w: user id", ErrInvalidInput)
	}
	var id int64
	if _, err := fmt.Sscan(args[0], &id); err != nil || id <= 0 {
		return 0, fmt.Errorf("%w: user id", ErrInvalidInput)
	}
	return id, nil
}

// parseUserAmount parses user ID and amount arguments.
func parseUserAmount(args []string) (userID, amount int64, err error) {
	if len(args) < 2 {
		return 0, 0, fmt.Errorf("%w: user id and amount", ErrInvalidInput)
	}
	if _, err := fmt.Sscan(args[0], &userID); err != nil || userID <= 0 {
		return 0, 0, fmt.Errorf("%w: user id", ErrInvalidInput)
	}
	if _, err := fmt.Sscan(args[1], &amount); err != nil || amount <= 0 {
		return 0, 0, fmt.Errorf("%w: amount", ErrInvalidInput)
	}
	return userID, amount, nil
}
