package models

import "time"

// User represents a Telegram bot user stored in SQLite.
type User struct {
	ID        int64
	Username  string
	FirstName string
	LastName  string
	Points    int64
	IsBanned  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
