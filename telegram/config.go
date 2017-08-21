package telegram

import (
	"encoding/json"
	"errors"
	"net/url"
	"time"
)

// API endpoint mask
const (
	APIEndpoint = "https://api.telegram.org/bot%s/%s"
)

// Errors
var (
	ErrAPINotOk     = errors.New("not ok")
	ErrAPIForbidden = errors.New("forbidden")
	ErrJobTimedOut  = errors.New("job request timed out")
)

// Config of bot
type Config struct {
	Token    string        `json:"token"`
	Cert     string        `json:"cert"`
	PKey     string        `json:"pkey"`
	Host     string        `json:"host"`
	Port     string        `json:"port"`
	PoolSize int           `json:"pool_size"`
	TimeOut  time.Duration `json:"timeout"`
}

// WebhookConfig struct
type WebhookConfig struct {
	URL      *url.URL
	Cert     string
	PoolSize int
}

// NewWebhookConfig is WebhookConfig c-tor
func NewWebhookConfig(host, port, token, cert string, poolSize int) (WebhookConfig, error) {
	url, err := url.Parse(host + ":" + port + "/" + token)
	if err != nil {
		return WebhookConfig{}, err
	}
	return WebhookConfig{
		URL:      url,
		Cert:     cert,
		PoolSize: poolSize,
	}, nil
}

// MessageConfig struct
type MessageConfig struct {
	chatID int64
	text   string
}

// NewMessageConfig is MessageConfig c-to
func NewMessageConfig(chatID int64, text string) MessageConfig {
	return MessageConfig{
		chatID: chatID,
		text:   text,
	}
}

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
	ID        int64  `json:"id"`
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
