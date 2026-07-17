package formatter

import (
	"strings"
	"testing"
	"time"
)

func TestEscapeModes(t *testing.T) {
	if got := EscapeHTML(`<a&">`); got != "&lt;a&amp;&quot;&gt;" {
		t.Fatalf("EscapeHTML = %q", got)
	}
	if got := EscapeMarkdownV2("a_b*c"); !strings.Contains(got, `\_`) || !strings.Contains(got, `\*`) {
		t.Fatalf("EscapeMarkdownV2 = %q", got)
	}
	if got := EscapePlain("ok\x00x"); strings.Contains(got, "\x00") {
		t.Fatalf("EscapePlain retained null: %q", got)
	}
}

func TestTemplates(t *testing.T) {
	f := HTML()
	out := f.SearchResult(SearchResult{Found: true, ID: "1", Phone: "123", Username: "user", Type: "id", Latency: time.Millisecond})
	if !strings.Contains(out, "<b>Search Result</b>") || !strings.Contains(out, "<code>1</code>") {
		t.Fatalf("SearchResult = %q", out)
	}
	if !strings.Contains(f.Profile(Profile{UserID: 7, Points: 3}), "7") {
		t.Fatal("Profile missing user id")
	}
	if !strings.Contains(f.History(nil), "No recent searches") {
		t.Fatal("empty history template mismatch")
	}
	if !strings.Contains(f.Statistics(Statistics{TotalSearches: 9}), "9") {
		t.Fatal("Statistics missing total")
	}
	if !strings.Contains(f.Admin(AdminPanel{Lines: []string{"ban 1"}}), "ban 1") {
		t.Fatal("Admin missing line")
	}
	if !strings.Contains(f.Error("boom"), "boom") || !strings.Contains(f.Success("done"), "done") {
		t.Fatal("Error/Success mismatch")
	}
	if f.ParseMode() != "HTML" {
		t.Fatalf("ParseMode = %q", f.ParseMode())
	}
}

func TestMarkdownV2SearchNotFound(t *testing.T) {
	f := MarkdownV2()
	out := f.SearchResult(SearchResult{Found: false})
	if !strings.Contains(out, "*Not found*") {
		t.Fatalf("got %q", out)
	}
}
