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
	ErrAPIKeybord   = errors.New("keybord setting error")
	ErrAPINoMessage = errors.New("not message")
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

// Keys bash (rus)
var Keys = []string{
	"случайные",
	"новые",
	"по рейтингу",
	"лучшие",
	"Бездна",
	"топ Бездны",
	"лучшие Бездны",
}

// Themes bash
var Themes = map[string]string{
	Keys[0]: "random",
	Keys[1]: "",
	Keys[2]: "byrating",
	Keys[3]: "best",
	Keys[4]: "abyss",
	Keys[5]: "abysstop",
	Keys[6]: "abyssbest",
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
