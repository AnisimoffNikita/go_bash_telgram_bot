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
	Token    string        `yaml:"token"`
	Cert     string        `yaml:"cert"`
	PKey     string        `yaml:"pkey"`
	Host     string        `yaml:"host"`
	Port     string        `yaml:"port"`
	PoolSize int           `yaml:"pool_size"`
	TimeOut  time.Duration `yaml:"timeout"`
	Debug    bool          `yaml:"debug"`
}

// Keys bash (rus)
var (
	Random    = "случайные"
	New       = "новые"
	ByRating  = "по рейтингу"
	Best      = "лучшие"
	Abyss     = "Бездна"
	AbyssTop  = "топ Бездны"
	AbyssBest = "лучшие Бездны"
)

// Other keys
var (
	Search   = "Поиск"
	Settings = "Настройки"
)

// Themes bash
var Themes = map[string]string{
	Random:    "random",
	New:       "",
	ByRating:  "byrating",
	Best:      "best",
	Abyss:     "abyss",
	AbyssTop:  "abysstop",
	AbyssBest: "abyssbest",
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
