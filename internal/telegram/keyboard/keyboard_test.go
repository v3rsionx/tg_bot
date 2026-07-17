package keyboard_test

import (
	"testing"

	"github.com/v3rsionx/tg_bot/internal/telegram/keyboard"
)

// TestInlineRowsBuildsCallbackButtons verifies inline keyboard construction.
func TestInlineRowsBuildsCallbackButtons(t *testing.T) {
	markup := keyboard.InlineRows([]keyboard.Button{{Text: "A", Data: "a"}})
	if len(markup.InlineKeyboard) != 1 || markup.InlineKeyboard[0][0].CallbackData != "a" {
		t.Fatalf("unexpected markup: %+v", markup)
	}
}

// TestReplyRowsBuildsTextButtons verifies reply keyboard construction.
func TestReplyRowsBuildsTextButtons(t *testing.T) {
	markup := keyboard.ReplyRows([]string{"Search"})
	if len(markup.Keyboard) != 1 || markup.Keyboard[0][0].Text != "Search" {
		t.Fatalf("unexpected markup: %+v", markup)
	}
}
