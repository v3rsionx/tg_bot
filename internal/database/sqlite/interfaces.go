package sqlite

import "github.com/v3rsionx/tg_bot/internal/repository"

// Compile-time checks that concrete types satisfy repository ports.
var (
	_ repository.UserRepository          = (*UserRepository)(nil)
	_ repository.TransactionRepository   = (*TransactionRepository)(nil)
	_ repository.SearchHistoryRepository = (*SearchHistoryRepository)(nil)
	_ repository.Transactor              = (*DatabaseManager)(nil)
)
