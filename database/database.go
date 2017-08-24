package database

import (
	"errors"
	"fmt"
	"log"
	"time"

	tarantool "github.com/tarantool/go-tarantool"
)

var db *tarantool.Connection

const (
	savedDB      = "tg_bot_saved"
	processorsDB = "tg_bot_processor"
	quoteDB      = "tg_bot_quotes"

	primary = "primary"
)

func init() {
	opts := tarantool.Opts{
		Timeout:       time.Second,
		Reconnect:     time.Second,
		MaxReconnects: 5,
		User:          "test",
		Pass:          "test",
	}

	var err error
	db, err = tarantool.Connect("localhost:3301", opts)

	if err != nil {
		log.Fatalf("Can't connect to tarantool: %s", err)
	}
}

// SetLastQuote func
func SetLastQuote(chatID int, quoteID string) error {
	return setString(quoteDB, chatID, quoteID)
}

// GetLastQuote func
func GetLastQuote(chatID int) (string, error) {
	return getString(quoteDB, chatID)
}

// RemoveLastQuote func
func RemoveLastQuote(chatID int) error {
	return deleteSpace(quoteDB, chatID)
}

// TruncateLastQuotes func
func TruncateLastQuotes() error {
	return truncateSpace(quoteDB)
}

// SetProcessor func
func SetProcessor(chatID int, processor string) error {
	return setString(processorsDB, chatID, processor)
}

// GetProcessor func
func GetProcessor(chatID int) (string, error) {
	return getString(processorsDB, chatID)
}

// RemoveProcessor func
func RemoveProcessor(chatID int) error {
	return deleteSpace(processorsDB, chatID)
}

// TruncateProcessor func
func TruncateProcessor() error {
	return truncateSpace(processorsDB)
}

func setString(name string, id int, str string) error {
	_, err := db.Upsert(name, []interface{}{id, str}, []interface{}{[]interface{}{"=", 1, str}})
	return err
}

func getString(name string, id int) (string, error) {
	resp, err := db.Select(name, primary, 0, 1, tarantool.IterEq, []interface{}{id})
	if err != nil {
		return "", err
	}

	if len(resp.Tuples()) == 0 {
		return "", errors.New("no processor")
	}

	processor, ok := resp.Tuples()[0][1].(string)
	if !ok {
		return "0", errors.New("incorrect field")
	}
	return processor, nil
}

func deleteSpace(name string, id int) error {
	_, err := db.Delete(name, primary, []interface{}{id})
	return err
}

func truncateSpace(name string) error {
	rawQuery := fmt.Sprintf("box.space.%s:truncate()", name)
	_, err := db.Eval(rawQuery, []interface{}{})
	return err
}
