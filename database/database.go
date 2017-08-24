package database

import (
	"fmt"
	"log"
	"time"

	tarantool "github.com/tarantool/go-tarantool"

	"github.com/AnisimoffNikita/go_bash_telgram_bot/helper"
)

var db *tarantool.Connection

const (
	processorsDB = "tg_bot_processor"
	savedDB      = "tg_bot_saved"
	quoteDB      = "tg_bot_quotes"
	searchDB     = "tg_bot_search"

	primary = "primary"
)

func init() {

	var config Config
	err := helper.GetYamlConfig(ConfigPath, &config)
	if err != nil {
		log.Fatal(err)
	}

	opts := tarantool.Opts{
		Timeout:       time.Duration(config.Timeout) * time.Second,
		Reconnect:     time.Duration(config.Reconnect) * time.Second,
		MaxReconnects: 5,
		User:          config.User,
		Pass:          config.Pass,
	}

	db, err = tarantool.Connect(fmt.Sprintf("%s:%s", config.Host, config.Port), opts)

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
		return "", ErrEmpty
	}

	processor, ok := resp.Tuples()[0][1].(string)
	if !ok {
		return "0", ErrIncorrectType
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

// SetSearch func
func SetSearch(chatID int, req string, index int, quoteID string) error {
	_, err := db.Upsert(searchDB, []interface{}{chatID, req, index, quoteID},
		[]interface{}{
			[]interface{}{"=", 1, req},
			[]interface{}{"=", 2, index},
			[]interface{}{"=", 3, quoteID}})
	return err
}

// GetSearch func
func GetSearch(chatID int) (string, int, string, error) {
	resp, err := db.Select(searchDB, primary, 0, 1, tarantool.IterEq, []interface{}{chatID})
	if err != nil {
		return "", -1, "", err
	}

	if len(resp.Tuples()) == 0 {
		return "", -1, "", ErrEmpty
	}

	processor, ok := resp.Tuples()[0][1].(string)
	if !ok {
		return "", -1, "", ErrIncorrectType
	}

	index, ok := resp.Tuples()[0][2].(uint64)
	if !ok {
		return "0", -1, "", ErrIncorrectType
	}
	quoteID, ok := resp.Tuples()[0][3].(string)
	if !ok {
		return "0", -1, "", ErrIncorrectType
	}

	return processor, int(index), quoteID, nil
}

// TruncateSearch func
func TruncateSearch() error {
	return truncateSpace(searchDB)
}

// SaveQuote func
func SaveQuote(chatID int, quoteID string) error {
	resp, err := db.Select(savedDB, primary, 0, 1, tarantool.IterEq, []interface{}{chatID})
	if err != nil {
		return err
	}

	if len(resp.Tuples()) == 0 {
		tuple := []interface{}{chatID, map[string]bool{quoteID: true}}
		resp, err = db.Insert(savedDB, tuple)
		return err
	}
	quotesRaw, ok := resp.Tuples()[0][1].(map[interface{}]interface{})
	if !ok {
		return ErrIncorrectType
	}

	quotes := make(map[string]bool)

	for kr, vr := range quotesRaw {
		k, okk := kr.(string)
		v, okv := vr.(bool)
		if !okk || !okv {
			return ErrIncorrectType
		}
		quotes[k] = v
	}
	quotes[quoteID] = true
	_, err = db.Update(savedDB, primary, []interface{}{chatID}, []interface{}{[]interface{}{"=", 1, quotes}})
	return err
}

// GetSavedQuotes func
func GetSavedQuotes(chatID int) (map[string]bool, error) {
	resp, err := db.Select(savedDB, primary, 0, 1, tarantool.IterEq, []interface{}{chatID})
	if err != nil {
		return nil, err
	}

	if len(resp.Tuples()) == 0 {
		return nil, ErrEmpty
	}

	quotesRaw, ok := resp.Tuples()[0][1].(map[interface{}]interface{})
	if !ok {
		return nil, ErrIncorrectType
	}

	quotes := make(map[string]bool)

	for kr, vr := range quotesRaw {
		k, okk := kr.(string)
		v, okv := vr.(bool)
		if !okk || !okv {
			return nil, ErrIncorrectType
		}
		quotes[k] = v
	}

	return quotes, nil
}

// DeleteSavedQuote func
func DeleteSavedQuote(chatID int, quoteID string) error {
	quotes, err := GetSavedQuotes(chatID)
	if err != nil {
		return err
	}

	delete(quotes, quoteID)
	_, err = db.Update(savedDB, primary, []interface{}{chatID}, []interface{}{[]interface{}{"=", 1, quotes}})
	return err
}

// TruncateSaved func
func TruncateSaved() error {
	return truncateSpace(savedDB)
}
