package handlers

import (
	"strings"
	"unicode"
)

// ParsedCommand is a normalized bot command extracted from message text.
type ParsedCommand struct {
	Name      string
	Args      []string
	Raw       string
	IsCommand bool
}

// ParseCommand extracts a command and arguments from Telegram message text.
func ParseCommand(text string) ParsedCommand {
	text = strings.TrimSpace(text)
	parsed := ParsedCommand{Raw: text}
	if text == "" || !strings.HasPrefix(text, "/") {
		return parsed
	}

	fields := strings.FieldsFunc(text, unicode.IsSpace)
	if len(fields) == 0 {
		return parsed
	}

	name := strings.TrimPrefix(fields[0], "/")
	if name == "" {
		return parsed
	}
	if at := strings.IndexByte(name, '@'); at >= 0 {
		name = name[:at]
	}
	name = strings.ToLower(name)
	if name == "" {
		return parsed
	}

	parsed.IsCommand = true
	parsed.Name = name
	if len(fields) > 1 {
		parsed.Args = append([]string(nil), fields[1:]...)
	}
	return parsed
}
