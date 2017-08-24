package database

import "errors"

//Database errors
var (
	ErrEmpty         = errors.New("empty")
	ErrIncorrectType = errors.New("incorrect type")
)
