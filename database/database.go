package database

import (
	"fmt"
	"time"

	tarantool "github.com/tarantool/go-tarantool"

	"github.com/AnisimoffNikita/go_bash_telgram_bot/helper"
)

// Tarantool connection type
type Tarantool struct {
	connection *tarantool.Connection
}

const (
	processorsDB = "tg_bot_processor"
	savedDB      = "tg_bot_saved"
	quoteDB      = "tg_bot_quotes"
	searchDB     = "tg_bot_search"

	primary = "primary"
)

// NewTarantool creates new tarantool connection
func NewTarantool() (*Tarantool, error) {

	var config Config
	err := helper.GetYamlConfig(ConfigPath, &config)
	if err != nil {
		return nil, fmt.Errorf("no tarantool config: %s", err)
	}

	opts := tarantool.Opts{
		Timeout:       time.Duration(config.Timeout) * time.Second,
		Reconnect:     time.Duration(config.Reconnect) * time.Second,
		MaxReconnects: config.MaxReconnects,
		User:          config.User,
		Pass:          config.Pass,
	}

	connection, err := tarantool.Connect(fmt.Sprintf("%s:%s", config.Host, config.Port), opts)

	if err != nil {
		return nil, fmt.Errorf("cannot connect to tarantool: %s", err)
	}

	rawQueryCreateSpace := "box.schema.space.create('%s')"
	rawQueryCreateIndex := "box.space.%s:create_index('%s', {type = 'hash',parts = {1, 'unsigned'}})"
	dbs := []string{processorsDB, savedDB, quoteDB, searchDB}

	for _, db := range dbs {
		space := connection.Schema.Spaces[db]
		if space == nil {
			_, err := connection.Eval(fmt.Sprintf(rawQueryCreateSpace, db), []interface{}{})
			if err != nil {
				return nil, fmt.Errorf("cannot create space %s: %s", db, err)
			}
			_, err = connection.Eval(fmt.Sprintf(rawQueryCreateIndex, db, primary), []interface{}{})
			if err != nil {
				return nil, fmt.Errorf("cannot create index on %s: %s", db, err)
			}
		}
	}

	return &Tarantool{connection: connection}, nil
}

// SetLastQuote func  (db *Tarantool)
func (db *Tarantool) SetLastQuote(chatID int, quoteID string) error {
	return db.setString(quoteDB, chatID, quoteID)
}

// GetLastQuote func  (db *Tarantool)
func (db *Tarantool) GetLastQuote(chatID int) (string, error) {
	return db.getString(quoteDB, chatID)
}

// RemoveLastQuote func  (db *Tarantool)
func (db *Tarantool) RemoveLastQuote(chatID int) error {
	return db.deleteSpace(quoteDB, chatID)
}

// TruncateLastQuotes func  (db *Tarantool)
func (db *Tarantool) TruncateLastQuotes() error {
	return db.truncateSpace(quoteDB)
}

// SetProcessor func  (db *Tarantool)
func (db *Tarantool) SetProcessor(chatID int, processor string) error {
	return db.setString(processorsDB, chatID, processor)
}

// GetProcessor func  (db *Tarantool)
func (db *Tarantool) GetProcessor(chatID int) (string, error) {
	return db.getString(processorsDB, chatID)
}

// RemoveProcessor func  (db *Tarantool)
func (db *Tarantool) RemoveProcessor(chatID int) error {
	return db.deleteSpace(processorsDB, chatID)
}

// TruncateProcessor func  (db *Tarantool)
func (db *Tarantool) TruncateProcessor() error {
	return db.truncateSpace(processorsDB)
}

func (db *Tarantool) setString(name string, id int, str string) error {
	_, err := db.connection.Upsert(name, []interface{}{id, str}, []interface{}{[]interface{}{"=", 1, str}})
	return err
}

func (db *Tarantool) getString(name string, id int) (string, error) {
	resp, err := db.connection.Select(name, primary, 0, 1, tarantool.IterEq, []interface{}{id})
	if err != nil {
		return "", err
	}

	if len(resp.Tuples()) == 0 {
		return "", ErrEmpty
	}

	if len(resp.Tuples()[0]) < 2 {
		return "", ErrEmpty
	}

	processor, ok := resp.Tuples()[0][1].(string)
	if !ok {
		return "0", ErrIncorrectType
	}
	return processor, nil
}

func (db *Tarantool) deleteSpace(name string, id int) error {
	_, err := db.connection.Delete(name, primary, []interface{}{id})
	return err
}

func (db *Tarantool) truncateSpace(name string) error {
	rawQuery := fmt.Sprintf("box.space.%s:truncate()", name)
	_, err := db.connection.Eval(rawQuery, []interface{}{})
	return err
}

// SetSearch func  (db *Tarantool)
func (db *Tarantool) SetSearch(chatID int, req string, index int, quoteID string) error {
	_, err := db.connection.Upsert(searchDB, []interface{}{chatID, req, index, quoteID},
		[]interface{}{
			[]interface{}{"=", 1, req},
			[]interface{}{"=", 2, index},
			[]interface{}{"=", 3, quoteID}})
	return err
}

// GetSearch func  (db *Tarantool)
func (db *Tarantool) GetSearch(chatID int) (string, int, string, error) {
	resp, err := db.connection.Select(searchDB, primary, 0, 1, tarantool.IterEq, []interface{}{chatID})
	if err != nil {
		return "", -1, "", err
	}

	if len(resp.Tuples()) == 0 {
		return "", -1, "", ErrEmpty
	}

	if len(resp.Tuples()[0]) < 4 {
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

// TruncateSearch func  (db *Tarantool)
func (db *Tarantool) TruncateSearch() error {
	return db.truncateSpace(searchDB)
}

// SaveQuote func  (db *Tarantool)
func (db *Tarantool) SaveQuote(chatID int, quoteID string) error {
	resp, err := db.connection.Select(savedDB, primary, 0, 1, tarantool.IterEq, []interface{}{chatID})
	if err != nil {
		return err
	}

	if len(resp.Tuples()) == 0 || len(resp.Tuples()[0]) < 2 {
		tuple := []interface{}{chatID, map[string]bool{quoteID: true}}
		resp, err = db.connection.Insert(savedDB, tuple)
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
	_, err = db.connection.Update(savedDB, primary, []interface{}{chatID}, []interface{}{[]interface{}{"=", 1, quotes}})
	return err
}

// GetSavedQuotes func  (db *Tarantool)
func (db *Tarantool) GetSavedQuotes(chatID int) (map[string]bool, error) {
	resp, err := db.connection.Select(savedDB, primary, 0, 1, tarantool.IterEq, []interface{}{chatID})
	if err != nil {
		return nil, err
	}

	if len(resp.Tuples()) == 0 {
		return nil, ErrEmpty
	}

	if len(resp.Tuples()[0]) < 2 {
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

// DeleteSavedQuote func  (db *Tarantool)
func (db *Tarantool) DeleteSavedQuote(chatID int, quoteID string) error {
	quotes, err := db.GetSavedQuotes(chatID)
	if err != nil {
		return err
	}

	delete(quotes, quoteID)
	_, err = db.connection.Update(savedDB, primary, []interface{}{chatID}, []interface{}{[]interface{}{"=", 1, quotes}})
	return err
}

// TruncateSaved func  (db *Tarantool)
func (db *Tarantool) TruncateSaved() error {
	return db.truncateSpace(savedDB)
}
