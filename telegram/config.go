package telegram

import (
	"errors"
	"net/url"
	"time"
)

// API endpoint mask
const (
	TelegramEndpoint = "https://api.telegram.org/bot%s/%s"
)

// Errors
var (
	ErrAPINoMessage = errors.New("not ok")
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
func newWebhookConfig(host, port, token, cert string, poolSize int) (WebhookConfig, error) {
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
