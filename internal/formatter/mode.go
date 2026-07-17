package formatter

import "github.com/v3rsi/tgbot-versionx/internal/constants"

// Mode selects output encoding.
type Mode int

const (
	ModePlain Mode = iota
	ModeHTML
	ModeMarkdownV2
)

// ParseMode returns the Telegram parse_mode string for m.
func (m Mode) ParseMode() string {
	switch m {
	case ModeHTML:
		return constants.ParseModeHTML
	case ModeMarkdownV2:
		return constants.ParseModeMarkdownV2
	default:
		return constants.ParseModePlain
	}
}

func (m Mode) escape(s string) string {
	switch m {
	case ModeHTML:
		return EscapeHTML(s)
	case ModeMarkdownV2:
		return EscapeMarkdownV2(s)
	default:
		return EscapePlain(s)
	}
}

func (m Mode) bold(s string) string {
	s = m.escape(s)
	switch m {
	case ModeHTML:
		return "<b>" + s + "</b>"
	case ModeMarkdownV2:
		return "*" + s + "*"
	default:
		return s
	}
}

func (m Mode) code(s string) string {
	s = m.escape(s)
	switch m {
	case ModeHTML:
		return "<code>" + s + "</code>"
	case ModeMarkdownV2:
		return "`" + s + "`"
	default:
		return s
	}
}

func (m Mode) italic(s string) string {
	s = m.escape(s)
	switch m {
	case ModeHTML:
		return "<i>" + s + "</i>"
	case ModeMarkdownV2:
		return "_" + s + "_"
	default:
		return s
	}
}
