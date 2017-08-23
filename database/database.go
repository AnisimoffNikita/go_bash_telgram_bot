package database

import (
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

// SaveQuote saves quote to db
func SaveQuote(chatID string, quoteID string) error {
	return nil
}

// DeleteQuote deletes quote to db
func DeleteQuote(chatID, quoteID string) error {
	return nil
}

func SetLastQuote(chatID, quoteID string) error {
	_, err := db.Insert(quoteDB, []interface{}{chatID, quoteID})
	return err
}

func HasLastQuote(chatID string) (bool, error) {
	resp, err := db.Select(quoteDB, primary, 0, 1, tarantool.IterEq, []interface{}{chatID})
}

func RemoveLastQuote(chatID string) error {
	_, err := db.Delete(quoteDB, primary, []interface{}{chatID})
	return err
}

func TruncateLastQuote() error {
	return truncateDB(quoteDB)
}

func SetNextProcessor(chatID, processor string) error {
	_, err := db.Insert(processorsDB, []interface{}{chatID, processor})
	return err
}

func RemoveNextProcessor(chatID string) error {
	_, err := db.Delete(processorsDB, primary, []interface{}{chatID})
	return err
}

func truncateDB(name string) error {
	rawQuery := fmt.Sprintf("box.space.%s:truncate()", name)
	_, err := db.Eval(rawQuery, []interface{}{})
	return err
}
