package telegram

import (
	"fmt"
	"time"
)

const (
	defaultShutdownTimeout = 10 * time.Second
	defaultHistoryLimit    = 10
)

// Config controls Telegram bot transport behavior.
type Config struct {
	// Token is the Telegram bot API token.
	Token string
	// OwnerIDs are privileged Telegram user IDs used by admin checks when
	// Authorizer implementations need them at construction time.
	OwnerIDs []int64
	// ShutdownTimeout bounds graceful shutdown cleanup.
	ShutdownTimeout time.Duration
	// HistoryLimit is the default number of history rows requested by /history.
	HistoryLimit int
}

// Validate checks Telegram transport configuration.
func (c Config) Validate() error {
	if c.Token == "" {
		return fmt.Errorf("telegram: Token is required")
	}
	if c.ShutdownTimeout < 0 {
		return fmt.Errorf("telegram: ShutdownTimeout must be >= 0")
	}
	if c.HistoryLimit < 0 {
		return fmt.Errorf("telegram: HistoryLimit must be >= 0")
	}
	return nil
}

// withDefaults returns a copy of Config with transport defaults applied.
func (c Config) withDefaults() Config {
	if c.ShutdownTimeout == 0 {
		c.ShutdownTimeout = defaultShutdownTimeout
	}
	if c.HistoryLimit == 0 {
		c.HistoryLimit = defaultHistoryLimit
	}
	return c
}

// errDependency returns a missing dependency error.
func errDependency(name string) error {
	return fmt.Errorf("telegram: Dependencies.%s is required", name)
}
