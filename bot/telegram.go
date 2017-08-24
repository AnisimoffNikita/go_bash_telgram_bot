package bot

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"

	yaml "gopkg.in/yaml.v2"

	"../bash"
	"../database"
	"../pool"
	"../telegram"
)

// Bot struct
type Bot struct {
	API        telegram.BotAPI
	Pool       *pool.Pool
	TimeOut    time.Duration
	Debug      bool
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

func newBot(token string, timeout time.Duration, poolSize int, debug bool) (*Bot, error) {
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

	bot.Debug = debug

	bot.Pool.Run()

	database.TruncateLastQuotes()
	database.TruncateProcessor()

	self, err := bot.API.GetMe()
	if err != nil {
		return nil, err
	}

	bot.API.Self = self
	return bot, nil
}

// StartBot begin bot work
func StartBot(configPath string) error {
	bytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("config found...")

	var config Config
	err = yaml.Unmarshal(bytes, &config)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("config correct...")

	bot, err := newBot(config.Token, config.TimeOut, config.PoolSize, config.Debug)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("connected...")

	defer database.TruncateLastQuotes()
	defer database.TruncateProcessor()
	defer database.TruncateSearch()

	if bot.Debug {
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
			_, err := bot.Pool.AddTaskSyncTimed(func() interface{} {
				processor, err := database.GetProcessor(update.Message.Chat.ID)

				if err != nil {
					return bot.Processors[DefaultProcessor](update)
				}
				return bot.Processors[processor](update)
			}, bot.TimeOut)

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
	_, err := bot.Pool.AddTaskSyncTimed(func() interface{} {
		bytes, _ := ioutil.ReadAll(r.Body)

		var update telegram.Update
		if err := json.Unmarshal(bytes, &update); err != nil {
			return err
		}

		processor, err := database.GetProcessor(update.Message.Chat.ID)
		if err != nil {
			return bot.Processors[DefaultProcessor](&update)
		}
		return bot.Processors[processor](&update)

	}, bot.TimeOut)

	if err != nil {
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
	database.TruncateLastQuotes()
	database.TruncateProcessor()
	database.SetProcessor(id, DefaultProcessor)

	buttons := telegram.NewReplyKeyboardMarkup([][]string{
		{Random},
		{Search},
		{Saved},
	})

	_, err := bot.API.SendTextWithKeybord(id, greeting, buttons)
	if err != nil {
		log.Println("can't send start messsage")
	}
	return err
}

func (bot *Bot) sendRandom(id int) error {
	quotes, err := bash.GetQuotes("random")
	if err != nil {
		log.Printf("can't get quotes: %s", err)
		return bot.start(id, WeHaveAnError)
	}

	buttons := telegram.NewReplyKeyboardMarkup([][]string{
		{Other},
		{Plus, Minus, Bayan},
		{Back},
	})

	quote := quotes[rand.Intn(len(quotes))]

	_, err = bot.API.SendTextWithKeybord(id, bash.QuoteToString(quote), buttons)
	if err != nil {
		log.Printf("can't send message %s", err)
		return bot.start(id, WeHaveAnError)
	}

	err = database.SetProcessor(id, RandomProcessor)
	if err != nil {
		log.Printf("can't set processor %s", err)
		return bot.start(id, WeHaveAnError)
	}
	err = database.SetLastQuote(id, quote.ID)
	if err != nil {
		log.Printf("can't set quote%s", err)
		return bot.start(id, WeHaveAnError)
	}

	return err
}

func (bot *Bot) feedbackQuote(update *telegram.Update) error {
	if update.Message == nil {
		return telegram.ErrAPINoMessage
	}

	text := update.Message.Text
	id := update.Message.Chat.ID

	lastQuote, err := database.GetLastQuote(id)
	if err != nil {
		log.Printf("can't get quote: %e", err)
		return bot.start(id, WeHaveAnError)
	}

	if text == Other {
		return bot.sendRandom(id)
	} else if text == Plus {
		go func() {
			err := database.SaveQuote(id, lastQuote)
			if err != nil {
				log.Printf("can't save quote: %e", err)
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
	database.SetProcessor(id, StartSearchProcessor)
	_, err := bot.API.SendTextWithoutKeybord(id, SearchReq)
	return err

}

func (bot *Bot) startSearch(update *telegram.Update) error {
	if update.Message == nil {
		return telegram.ErrAPINoMessage
	}

	text := update.Message.Text
	id := update.Message.Chat.ID

	err := database.SetSearch(id, text, 0, "")
	if err != nil {
		log.Printf("can't set search %s", err)
		return bot.start(id, WeHaveAnError)
	}
	err = database.SetProcessor(id, SearchProcessor)
	if err != nil {
		log.Printf("can't set processor%s", err)
		return bot.start(id, WeHaveAnError)
	}
	return bot.sendFound(id, text, 0)
}

func (bot *Bot) feedbackSearch(update *telegram.Update) error {

	if update.Message == nil {
		log.Printf("feedbackSaved error: %s", telegram.ErrAPINoMessage)
		return bot.start(update.Message.Chat.ID, WeHaveAnError)
	}

	id := update.Message.Chat.ID

	req, index, quote, err := database.GetSearch(id)
	if err != nil {
		log.Printf("can't get search: %s", err)
		return bot.start(id, WeHaveAnError)
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
		log.Printf("can't search message: %s", err)
		return bot.start(id, WeHaveAnError)
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
		log.Printf("can't send message %s", err)
		return bot.start(id, WeHaveAnError)
	}

	err = database.SetSearch(id, text, index+1, quote.ID)
	if err != nil {
		log.Printf("can't set quote%s", err)
		return bot.start(id, WeHaveAnError)
	}
	return err
}

func (bot *Bot) sendSaved(id int) error {
	quotes, err := database.GetSavedQuotes(id)
	if err == database.ErrEmpty {
		return bot.start(id, NothingToSend)
	}

	if err != nil {
		log.Printf("sendSaved error: %s", err)
		return bot.start(id, WeHaveAnError)
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
		log.Printf("can't get quote by id%s", err)
		return bot.start(id, WeHaveAnError)
	}

	_, err = bot.API.SendTextWithKeybord(id, bash.QuoteToString(quote), buttons)
	if err != nil {
		log.Printf("can't send message %s", err)
		return bot.start(id, WeHaveAnError)
	}

	err = database.SetLastQuote(id, quote.ID)
	if err != nil {
		log.Printf(" can't set quote %s", err)
		return bot.start(id, WeHaveAnError)
	}

	err = database.SetProcessor(id, SaveProcessor)
	if err != nil {
		log.Printf(" can't set processor: %s", err)
		return bot.start(id, WeHaveAnError)
	}

	return err
}

func (bot *Bot) feedbackSaved(update *telegram.Update) error {
	if update.Message == nil {
		log.Printf("feedbackSaved error: %s", telegram.ErrAPINoMessage)
		return bot.start(update.Message.Chat.ID, WeHaveAnError)
	}

	text := update.Message.Text
	id := update.Message.Chat.ID

	lastQuote, err := database.GetLastQuote(id)
	if err != nil {
		log.Printf("feedbackSaved error: %s", err)
		return bot.start(id, WeHaveAnError)
	}

	if text == Other {
		return bot.sendSaved(id)
	} else if text == Delete {

		err := database.DeleteSavedQuote(id, lastQuote)
		if err != nil {
			log.Printf("can't save quote: %s", err)
		}
		return bot.sendSaved(id)
	} else if text == Back {
		return bot.start(id, NothingToSend)
	}
	return bot.start(id, BadThing)
}
