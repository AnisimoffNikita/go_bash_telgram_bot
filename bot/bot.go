package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/AnisimoffNikita/go_bash_telgram_bot/bash"
	"github.com/AnisimoffNikita/go_bash_telgram_bot/database"
	"github.com/AnisimoffNikita/go_bash_telgram_bot/helper"
	"github.com/AnisimoffNikita/go_bash_telgram_bot/pool"
	"github.com/AnisimoffNikita/go_bash_telgram_bot/telegram"
)

// Bot struct
type Bot struct {
	API        telegram.BotAPI
	Pool       *pool.Pool
	DB         *database.Tarantool
	TimeOut    time.Duration
	Processors map[string]func(update *telegram.Update) error
}

// Processors name
const (
	DefaultProcessor     = "default"
	RandomProcessor      = "random"
	SaveProcessor        = "save"
	StartSearchProcessor = "startSearch"
	SearchProcessor      = "search"
)

func newBot(token string, timeout time.Duration, poolSize int) (*Bot, error) {
	bot := &Bot{
		API: telegram.BotAPI{
			Token:  token,
			Client: &http.Client{},
		},
		Pool:    pool.NewPool(poolSize),
		TimeOut: timeout * time.Millisecond,
	}

	bot.Processors = map[string]func(update *telegram.Update) error{
		DefaultProcessor:     bot.processUpdate,
		RandomProcessor:      bot.feedbackQuote,
		StartSearchProcessor: bot.startSearch,
		SearchProcessor:      bot.feedbackSearch,
		SaveProcessor:        bot.feedbackSaved,
	}

	var err error
	bot.DB, err = database.NewTarantool()

	if err != nil {
		return nil, err
	}

	bot.Pool.Run()

	bot.DB.TruncateLastQuotes()
	bot.DB.TruncateProcessor()

	self, err := bot.API.GetMe()
	if err != nil {
		return nil, err
	}

	bot.API.Self = self
	return bot, nil
}

// StartBot begin bot work
func StartBot() error {
	var config Config
	err := helper.GetYamlConfig(configPath, &config)
	if err != nil {
		log.Fatal(err)
	}

	bot, err := newBot(config.Token, config.TimeOut, config.PoolSize)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("connected...")

	defer bot.DB.TruncateLastQuotes()
	defer bot.DB.TruncateProcessor()
	defer bot.DB.TruncateSearch()

	if config.PKey == "" {
		log.Println("start (debug)...")
		return bot.processUpdatesChannel(config.PoolSize)
	}

	webhookConfig, err := newWebhookConfig(config.Host,
		config.Port,
		config.Token,
		config.Cert,
		config.PoolSize)

	if err != nil {
		log.Fatal(err)
	}
	_, err = bot.setWebhook(&webhookConfig)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("webhook set...")
	log.Println("start...")

	http.HandleFunc("/"+bot.API.Token, bot.updateHandler)
	return http.ListenAndServeTLS("0.0.0.0:"+config.Port,
		config.Cert,
		config.PKey,
		nil)
}

func (bot *Bot) processUpdatesChannel(channelSize int) error {
	updates, err := bot.getUpdatesChannel(channelSize)
	if err != nil {
		return err
	}

	for update := range updates {
		go func(update *telegram.Update) {
			res, err := bot.Pool.AddTaskSyncTimed(func() interface{} {
				processor, err := bot.DB.GetProcessor(update.Message.Chat.ID)

				if err != nil {
					return bot.Processors[DefaultProcessor](update)
				}
				return bot.Processors[processor](update)
			}, bot.TimeOut)

			if res != nil {
				log.Println(err)
			}
			if err != nil {
				log.Println(err)
			}
		}(update)
	}
	return nil
}

func (bot *Bot) getUpdatesChannel(poolSize int) (<-chan *telegram.Update, error) {
	updatesChannel := make(chan *telegram.Update, poolSize)
	offset := -1

	go func() {
		for {
			time.Sleep(100)

			params := url.Values{}
			if offset != -1 {
				params.Add("offset", strconv.Itoa(offset))
			}

			resp, err := bot.API.MakeRequest("getUpdates", params)
			if err != nil {
				log.Println(err)
				continue
			}
			var updates []*telegram.Update

			err = json.Unmarshal(resp.Result, &updates)
			if err != nil {
				log.Println(err)
				continue
			}
			for _, v := range updates {
				offset = v.UpdateID + 1
				updatesChannel <- v
			}
		}
	}()

	return updatesChannel, nil
}

func (bot *Bot) updateHandler(w http.ResponseWriter, r *http.Request) {
	res, err := bot.Pool.AddTaskSyncTimed(func() interface{} {
		bytes, _ := ioutil.ReadAll(r.Body)

		var update telegram.Update
		if err := json.Unmarshal(bytes, &update); err != nil {
			return err
		}

		processor, err := bot.DB.GetProcessor(update.Message.Chat.ID)
		if err != nil {
			return bot.Processors[DefaultProcessor](&update)
		}
		return bot.Processors[processor](&update)

	}, bot.TimeOut)

	if err != nil {
		log.Println(err)
	}
	if res != nil {
		log.Println(err)
	}
}

func (bot *Bot) setWebhook(webhookConfig *WebhookConfig) (telegram.APIResponse, error) {
	params := make(map[string]string)
	params["url"] = webhookConfig.URL.String()
	params["max_connections"] = strconv.Itoa(int(webhookConfig.PoolSize))

	resp, err := bot.API.UploadFile("setWebhook", params, "certificate", webhookConfig.Cert)

	if err != nil {
		return telegram.APIResponse{}, err
	}

	return resp, nil
}

func (bot *Bot) processUpdate(update *telegram.Update) error {

	if update.Message == nil {
		return telegram.ErrAPINoMessage
	}

	text := update.Message.Text
	id := update.Message.Chat.ID

	if text == Start {
		return bot.start(id, WhatSend)
	} else if text == Random {
		return bot.sendRandom(id)
	} else if text == Search {
		return bot.sendSearch(id)
	} else if text == Saved {
		return bot.sendSaved(id)
	}
	return bot.start(id, BadThing)
}

func (bot *Bot) start(id int, greeting string) error {
	bot.DB.TruncateLastQuotes()
	bot.DB.TruncateProcessor()
	bot.DB.SetProcessor(id, DefaultProcessor)

	buttons := telegram.NewReplyKeyboardMarkup([][]string{
		{Random},
		{Search},
		{Saved},
	})

	_, err := bot.API.SendTextWithKeybord(id, greeting, buttons)
	if err != nil {
		return fmt.Errorf("can't send start messsage: %s", err)
	}
	return nil
}

func (bot *Bot) sendRandom(id int) error {
	quotes, err := bash.GetQuotes("random")
	if err != nil {
		return fmt.Errorf("can't get quotes: %s", err)
	}

	buttons := telegram.NewReplyKeyboardMarkup([][]string{
		{Other},
		{Plus, Minus, Bayan},
		{Back},
	})

	quote := quotes[rand.Intn(len(quotes))]

	_, err = bot.API.SendTextWithKeybord(id, bash.QuoteToString(quote), buttons)
	if err != nil {
		return fmt.Errorf("can't send message %s", err)
	}

	err = bot.DB.SetProcessor(id, RandomProcessor)
	if err != nil {
		return fmt.Errorf("can't set processor %s", err)
	}
	err = bot.DB.SetLastQuote(id, quote.ID)
	if err != nil {
		return fmt.Errorf("can't set quote%s", err)
	}

	return nil
}

func (bot *Bot) feedbackQuote(update *telegram.Update) error {
	if update.Message == nil {
		return telegram.ErrAPINoMessage
	}

	text := update.Message.Text
	id := update.Message.Chat.ID

	lastQuote, err := bot.DB.GetLastQuote(id)
	if err != nil {
		return fmt.Errorf("can't get quote: %s", err)
	}

	if text == Other {
		return bot.sendRandom(id)
	} else if text == Plus {
		go func() {
			err := bot.DB.SaveQuote(id, lastQuote)
			if err != nil {
				log.Printf("can't save quote: %s", err)
			}
		}()
		go bash.Plus(lastQuote)
		return bot.sendRandom(id)
	} else if text == Minus {
		go bash.Minus(lastQuote)
		return bot.sendRandom(id)
	} else if text == Bayan {
		go bash.Bayan(lastQuote)
		return bot.sendRandom(id)
	} else if text == Back {
		return bot.start(id, WhatSend)
	}
	return bot.start(id, BadThing)
}

func (bot *Bot) sendSearch(id int) error {
	err := bot.DB.SetProcessor(id, StartSearchProcessor)
	if err != nil {
		return fmt.Errorf("can't set processor %s", err)
	}
	_, err = bot.API.SendTextWithoutKeybord(id, SearchReq)
	if err != nil {
		return fmt.Errorf("can't send message %s", err)
	}
	return err

}

func (bot *Bot) startSearch(update *telegram.Update) error {
	if update.Message == nil {
		return telegram.ErrAPINoMessage
	}

	text := update.Message.Text
	id := update.Message.Chat.ID

	err := bot.DB.SetSearch(id, text, 0, "")
	if err != nil {
		return fmt.Errorf("can't set search %s", err)
	}
	err = bot.DB.SetProcessor(id, SearchProcessor)
	if err != nil {
		return fmt.Errorf("can't set processor%s", err)
	}
	return bot.sendFound(id, text, 0)
}

func (bot *Bot) feedbackSearch(update *telegram.Update) error {

	if update.Message == nil {
		return fmt.Errorf("feedbackSaved error: %s", telegram.ErrAPINoMessage)
	}

	id := update.Message.Chat.ID

	req, index, quote, err := bot.DB.GetSearch(id)
	if err != nil {
		return fmt.Errorf("can't get search: %s", err)
	}

	text := update.Message.Text

	if text == Other {
		return bot.sendFound(id, req, index)
	} else if text == Plus {
		go bash.Plus(quote)
		return bot.sendFound(id, req, index)
	} else if text == Minus {
		go bash.Minus(quote)
		return bot.sendFound(id, req, index)
	} else if text == Bayan {
		go bash.Bayan(quote)
		return bot.sendFound(id, req, index)
	} else if text == Back {
		return bot.start(id, WhatSend)
	}
	return bot.start(id, BadThing)

}

func (bot *Bot) sendFound(id int, text string, index int) error {

	quotes, err := bash.Search(text)
	if err != nil {
		return fmt.Errorf("can't search message: %s", err)
	}

	buttons := telegram.NewReplyKeyboardMarkup([][]string{
		{Other},
		{Plus, Minus, Bayan},
		{Back},
	})

	if len(quotes) == 0 || len(quotes) <= index {
		return bot.start(id, NothingToSend)
	}

	quote := quotes[index]
	_, err = bot.API.SendTextWithKeybord(id, bash.QuoteToString(quote), buttons)
	if err != nil {
		return fmt.Errorf("can't send message %s", err)
	}

	err = bot.DB.SetSearch(id, text, index+1, quote.ID)
	if err != nil {
		return fmt.Errorf("can't set quote%s", err)
	}
	return err
}

func (bot *Bot) sendSaved(id int) error {
	quotes, err := bot.DB.GetSavedQuotes(id)
	if err == database.ErrEmpty {
		return bot.start(id, NothingToSend)
	}

	if err != nil {
		return fmt.Errorf("sendSaved error: %s", err)
	}

	buttons := telegram.NewReplyKeyboardMarkup([][]string{
		{Other},
		{Delete},
		{Back},
	})

	l := len(quotes)
	if l <= 0 {
		return bot.start(id, NothingToSend)
	}
	n := rand.Intn(l)
	i := 0
	var quoteID string
	for k := range quotes {
		if i == n {
			quoteID = k
			break
		}
		i++
	}

	quote, err := bash.GetQuoteByID(quoteID)
	if err != nil {
		return fmt.Errorf("can't get quote by id: %s", err)
	}

	_, err = bot.API.SendTextWithKeybord(id, bash.QuoteToString(quote), buttons)
	if err != nil {
		return fmt.Errorf("can't send message: %s", err)
	}

	err = bot.DB.SetLastQuote(id, quote.ID)
	if err != nil {
		return fmt.Errorf("can't set quote: %s", err)
	}

	err = bot.DB.SetProcessor(id, SaveProcessor)
	if err != nil {
		return fmt.Errorf("can't set processor: %s", err)
	}

	return err
}

func (bot *Bot) feedbackSaved(update *telegram.Update) error {
	if update.Message == nil {
		return fmt.Errorf("feedbackSaved error: %s", telegram.ErrAPINoMessage)
	}

	text := update.Message.Text
	id := update.Message.Chat.ID

	lastQuote, err := bot.DB.GetLastQuote(id)
	if err != nil {
		return fmt.Errorf("feedbackSaved error: %s", err)
	}

	if text == Other {
		return bot.sendSaved(id)
	} else if text == Delete {

		err := bot.DB.DeleteSavedQuote(id, lastQuote)
		if err != nil {
			return fmt.Errorf("can't save quote: %s", err)
		}
		return bot.sendSaved(id)
	} else if text == Back {
		return bot.start(id, NothingToSend)
	}
	return bot.start(id, BadThing)
}
