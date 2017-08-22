package telegram

import "encoding/json"

// User type telegram
type User struct {
	ID           int    `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	UserName     string `json:"username"`
	LanguageCode string `json:"language_code"`
}

// APIResponse type telegram
type APIResponse struct {
	Ok          bool                `json:"ok"`
	Result      json.RawMessage     `json:"result"`
	ErrorCode   int                 `json:"error_code"`
	Description string              `json:"description"`
	Parameters  *ResponseParameters `json:"parameters"`
}

// ResponseParameters type telegram
type ResponseParameters struct {
	MigrateToChatID int64 `json:"migrate_to_chat_id"`
	RetryAfter      int   `json:"retry_after"`
}

// Update type telegram
type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message"`
}

// Chat type telegram
type Chat struct {
	ID        int    `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	UserName  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// Message type telegram
type Message struct {
	MessageID int    `json:"message_id"`
	From      *User  `json:"from"`
	Date      int    `json:"date"`
	Chat      *Chat  `json:"chat"`
	Text      string `json:"text"` // optional
}

// ReplyKeyboardMarkup type telegram
type ReplyKeyboardMarkup struct {
	Keyboard        [][]KeyboardButton `json:"keyboard"`
	ResizeKeyboard  bool               `json:"resize_keyboard"`
	OneTimeKeyboard bool               `json:"one_time_keyboard"`
	Selective       bool               `json:"selective"`
}

// ReplyKeyboardRemove type telegram
type ReplyKeyboardRemove struct {
	RemoveKeyboard bool `json:"remove_keyboard"`
	Selective      bool `json:"selective"`
}

// KeyboardButton type telegram
type KeyboardButton struct {
	Text string `json:"text"`
}

func newReplyKeyboardMarkup(text [][]string) ReplyKeyboardMarkup {

	buttons := make([][]KeyboardButton, len(text))
	for i, v := range text {
		buttons[i] = make([]KeyboardButton, len(v))
		for j := range v {
			buttons[i][j].Text = text[i][j]
		}
	}

	keybord := ReplyKeyboardMarkup{
		Keyboard:        buttons,
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
		Selective:       true,
	}

	return keybord
}
