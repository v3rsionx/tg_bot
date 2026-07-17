package formatter

import (
	"fmt"
	"strings"
	"time"
)

// Formatter renders reusable message templates.
// Construct via New; no package-level mutable state.
type Formatter struct {
	mode Mode
}

// New constructs a Formatter for the given mode.
func New(mode Mode) *Formatter {
	return &Formatter{mode: mode}
}

// Mode returns the active output mode.
func (f *Formatter) Mode() Mode {
	if f == nil {
		return ModePlain
	}
	return f.mode
}

// ParseMode returns the Telegram parse_mode for this formatter.
func (f *Formatter) ParseMode() string {
	return f.Mode().ParseMode()
}

// Escape escapes text for the active mode.
func (f *Formatter) Escape(s string) string {
	return f.Mode().escape(s)
}

// SearchResult formats a search result template.
func (f *Formatter) SearchResult(r SearchResult) string {
	m := f.Mode()
	if !r.Found {
		return m.bold("Not found") + "\n" + m.escape("No matching record.")
	}
	var b strings.Builder
	b.WriteString(m.bold("Search Result"))
	b.WriteByte('\n')
	if r.Type != "" {
		b.WriteString(m.escape("Type: ") + m.code(r.Type) + "\n")
	}
	if r.ID != "" {
		b.WriteString(m.escape("ID: ") + m.code(r.ID) + "\n")
	}
	if r.Phone != "" {
		b.WriteString(m.escape("Phone: ") + m.code(r.Phone) + "\n")
	}
	if r.Username != "" {
		b.WriteString(m.escape("Username: ") + m.code("@"+strings.TrimPrefix(r.Username, "@")) + "\n")
	}
	if r.Latency > 0 {
		b.WriteString(m.escape("Latency: ") + m.escape(r.Latency.String()))
	}
	return strings.TrimRight(b.String(), "\n")
}

// Profile formats a user profile template.
func (f *Formatter) Profile(p Profile) string {
	m := f.Mode()
	var b strings.Builder
	b.WriteString(m.bold("Profile") + "\n")
	b.WriteString(m.escape("User ID: ") + m.code(fmt.Sprintf("%d", p.UserID)) + "\n")
	if p.Username != "" {
		b.WriteString(m.escape("Username: ") + m.code("@"+strings.TrimPrefix(p.Username, "@")) + "\n")
	}
	b.WriteString(m.escape("Points: ") + m.code(fmt.Sprintf("%d", p.Points)) + "\n")
	status := "active"
	if p.Banned {
		status = "banned"
	}
	b.WriteString(m.escape("Status: ") + m.escape(status))
	if !p.JoinedAt.IsZero() {
		b.WriteByte('\n')
		b.WriteString(m.escape("Joined: ") + m.escape(p.JoinedAt.UTC().Format(time.RFC3339)))
	}
	return b.String()
}

// History formats a history list template.
func (f *Formatter) History(entries []HistoryEntry) string {
	m := f.Mode()
	if len(entries) == 0 {
		return m.bold("History") + "\n" + m.escape("No recent searches.")
	}
	var b strings.Builder
	b.WriteString(m.bold("History") + "\n")
	for i, e := range entries {
		found := "miss"
		if e.Found {
			found = "hit"
		}
		line := fmt.Sprintf("%d. %s (%s) — %s", i+1, e.Query, e.Type, found)
		if !e.CreatedAt.IsZero() {
			line += " @ " + e.CreatedAt.UTC().Format("2006-01-02 15:04")
		}
		b.WriteString(m.escape(line))
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

// Statistics formats a statistics template.
func (f *Formatter) Statistics(s Statistics) string {
	m := f.Mode()
	var b strings.Builder
	b.WriteString(m.bold("Statistics") + "\n")
	b.WriteString(m.escape(fmt.Sprintf("Total searches: %d", s.TotalSearches)) + "\n")
	b.WriteString(m.escape(fmt.Sprintf("Successful: %d", s.SuccessfulSearches)) + "\n")
	b.WriteString(m.escape(fmt.Sprintf("Failed: %d", s.FailedSearches)) + "\n")
	b.WriteString(m.escape(fmt.Sprintf("Today: %d", s.TodaySearches)) + "\n")
	b.WriteString(m.escape(fmt.Sprintf("Current users: %d", s.CurrentUsers)) + "\n")
	b.WriteString(m.escape(fmt.Sprintf("Avg latency: %s", s.AverageLatency)) + "\n")
	b.WriteString(m.escape(fmt.Sprintf("Peak latency: %s", s.PeakLatency)) + "\n")
	b.WriteString(m.escape(fmt.Sprintf("Cache hit rate: %.1f%%", s.CacheHitRate*100)))
	return b.String()
}

// Admin formats an admin panel template.
func (f *Formatter) Admin(a AdminPanel) string {
	m := f.Mode()
	title := a.Title
	if title == "" {
		title = "Admin"
	}
	var b strings.Builder
	b.WriteString(m.bold(title) + "\n")
	for _, line := range a.Lines {
		b.WriteString(m.escape(line) + "\n")
	}
	if a.Footer != "" {
		b.WriteString(m.italic(a.Footer))
	}
	return strings.TrimRight(b.String(), "\n")
}

// Error formats an error template.
func (f *Formatter) Error(message string) string {
	m := f.Mode()
	if message == "" {
		message = "Something went wrong."
	}
	return m.bold("Error") + "\n" + m.escape(message)
}

// Success formats a success template.
func (f *Formatter) Success(message string) string {
	m := f.Mode()
	if message == "" {
		message = "OK"
	}
	return m.bold("Success") + "\n" + m.escape(message)
}

// Plain returns a plain-text formatter.
func Plain() *Formatter { return New(ModePlain) }

// HTML returns an HTML formatter.
func HTML() *Formatter { return New(ModeHTML) }

// MarkdownV2 returns a MarkdownV2 formatter.
func MarkdownV2() *Formatter { return New(ModeMarkdownV2) }
