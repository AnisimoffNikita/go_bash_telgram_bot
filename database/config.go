package database

import (
	"errors"
)

//Database errors
var (
	ErrEmpty         = errors.New("empty")
	ErrIncorrectType = errors.New("incorrect type")
)

// Path to db config
const (
	ConfigPath = "db.yml"
)

// Config of db
type Config struct {
	Host          string `yaml:"host"`
	Port          string `yaml:"port"`
	User          string `yaml:"user"`
	Pass          string `yaml:"pass"`
	Timeout       int    `yaml:"timeout"`
	Reconnect     int    `yaml:"reconnect"`
	MaxReconnects uint   `yaml:"max_reconnects"`
}
