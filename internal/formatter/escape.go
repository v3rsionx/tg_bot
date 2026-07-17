package formatter

import (
	"strings"
	"unicode/utf8"
)

// EscapeHTML escapes text for Telegram HTML parse mode.
func EscapeHTML(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	)
	return replacer.Replace(s)
}

// EscapeMarkdownV2 escapes text for Telegram MarkdownV2 parse mode.
func EscapeMarkdownV2(s string) string {
	const specials = "_*[]()~`>#+-=|{}.!\\"
	var b strings.Builder
	b.Grow(len(s) + 8)
	for _, r := range s {
		if strings.ContainsRune(specials, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// EscapePlain returns text with invalid UTF-8 replaced and control noise removed.
func EscapePlain(s string) string {
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "�")
	}
	return strings.ReplaceAll(s, "\x00", "")
}
