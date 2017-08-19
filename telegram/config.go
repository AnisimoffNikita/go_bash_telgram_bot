package telegram

import (
	"errors"
	"net/url"
)

const (
	APIEndpoint = "https://api.telegram.org/bot%s/%s"
)

var (
	ErrAPIForbidden = errors.New("forbidden")
	ErrJobTimedOut  = errors.New("job request timed out")
)

type Config struct {
	Token    string `json:"token"`
	Cert     string `json:"cert"`
	PKey     string `json:"pkey"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	PoolSize int32  `json:"pool_size"`
}

type WebhookConfig struct {
	URL      *url.URL
	Cert     string
	PoolSize int32
}

func NewWebhookConfig(host, port, cert string, poolSize int32) (WebhookConfig, error) {
	url, err := url.Parse(host + ":" + port)
	if err != nil {
		return WebhookConfig{}, err
	}
	return WebhookConfig{
		URL:      url,
		Cert:     cert,
		PoolSize: poolSize,
	}, nil
}
