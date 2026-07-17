package keyboard

import "github.com/go-telegram/bot/models"

// Button is a simple inline keyboard button descriptor.
type Button struct {
	Text string
	Data string
}

// InlineRows builds an inline keyboard markup from button rows.
func InlineRows(rows ...[]Button) *models.InlineKeyboardMarkup {
	keyboard := make([][]models.InlineKeyboardButton, 0, len(rows))
	for _, row := range rows {
		buttons := make([]models.InlineKeyboardButton, 0, len(row))
		for _, button := range row {
			buttons = append(buttons, models.InlineKeyboardButton{
				Text:         button.Text,
				CallbackData: button.Data,
			})
		}
		keyboard = append(keyboard, buttons)
	}
	return &models.InlineKeyboardMarkup{InlineKeyboard: keyboard}
}

// ReplyRows builds a reply keyboard markup from text rows.
func ReplyRows(rows ...[]string) *models.ReplyKeyboardMarkup {
	keyboard := make([][]models.KeyboardButton, 0, len(rows))
	for _, row := range rows {
		buttons := make([]models.KeyboardButton, 0, len(row))
		for _, text := range row {
			buttons = append(buttons, models.KeyboardButton{Text: text})
		}
		keyboard = append(keyboard, buttons)
	}
	return &models.ReplyKeyboardMarkup{
		Keyboard:       keyboard,
		ResizeKeyboard: true,
	}
}

// RemoveReply returns markup that removes a custom reply keyboard.
func RemoveReply() *models.ReplyKeyboardRemove {
	return &models.ReplyKeyboardRemove{RemoveKeyboard: true}
}
