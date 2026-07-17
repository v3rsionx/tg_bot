package constants

// Telegram bot commands (without leading slash).
const (
	CommandStart   = "start"
	CommandHelp    = "help"
	CommandProfile = "profile"
	CommandHistory = "history"
	CommandAdmin   = "admin"
)

// TelegramParseMode values.
const (
	ParseModeHTML       = "HTML"
	ParseModeMarkdownV2 = "MarkdownV2"
	ParseModePlain      = ""
)

// Telegram message limits.
const (
	TelegramMaxMessageBytes = 4096
	TelegramMaxCaptionBytes = 1024
)
