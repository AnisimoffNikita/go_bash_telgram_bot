package bot

import (
	"net/url"
	"time"
)

// API endpoint mask
const (
	TelegramEndpoint = "https://api.telegram.org/bot%s/%s"
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

// Menu constants
const (
	Start  = "/start"
	Random = "Случайную"
	Search = "Поиск"
	Saved  = "Сохранненые"
	Plus   = "➕"
	Minus  = "➖"
	Bayan  = "[ : ||| : ]"
	Other  = "Еще одну"
	Back   = "Назад"
)

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
