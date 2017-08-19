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
