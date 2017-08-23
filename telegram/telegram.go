package telegram

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
	"../pool"
)

// Bot struct
type Bot struct {
	API             BotAPI
	Pool            *pool.Pool
	TimeOut         time.Duration
	Debug           bool
	NextProcessor   func(update *Update) error
	LastQuote       bash.Quote
	SearchRequest   string
	LastSearchIndex int
}

func newBot(token string, timeout time.Duration, poolSize int, debug bool) (*Bot, error) {
	bot := &Bot{
		API: BotAPI{
			Token:  token,
			Client: &http.Client{},
		},
		Pool:          pool.NewPool(poolSize),
		TimeOut:       timeout * time.Millisecond,
		NextProcessor: nil,
	}

	bot.Debug = debug

	bot.Pool.Run()

	self, err := bot.API.getMe()
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
		go func(update *Update) {
			_, err := bot.Pool.AddTaskSyncTimed(func() interface{} {
				if bot.NextProcessor != nil {
					return bot.NextProcessor(update)
				}
				return bot.processUpdate(update)
			}, bot.TimeOut)

			if err != nil {
				log.Println(err)
			}
		}(update)
	}
	return nil
}

func (bot *Bot) getUpdatesChannel(poolSize int) (<-chan *Update, error) {
	updatesChannel := make(chan *Update, poolSize)
	offset := -1

	go func() {
		for {
			time.Sleep(100)

			params := url.Values{}
			if offset != -1 {
				params.Add("offset", strconv.Itoa(offset))
			}

			resp, err := bot.API.makeRequest("getUpdates", params)
			if err != nil {
				log.Println(err)
				continue
			}
			var updates []*Update

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

		var update Update
		if err := json.Unmarshal(bytes, &update); err != nil {
			return err
		}

		return bot.processUpdate(&update)

	}, bot.TimeOut)

	if err != nil {
		log.Println(err)
	}
}

func (bot *Bot) setWebhook(webhookConfig *WebhookConfig) (APIResponse, error) {
	params := make(map[string]string)
	params["url"] = webhookConfig.URL.String()
	params["max_connections"] = strconv.Itoa(int(webhookConfig.PoolSize))

	resp, err := bot.API.uploadFile("setWebhook", params, "certificate", webhookConfig.Cert)

	if err != nil {
		return APIResponse{}, err
	}

	return resp, nil
}

func (bot *Bot) processUpdate(update *Update) error {
	bot.NextProcessor = nil

	if update.Message == nil {
		return ErrAPINoMessage
	}

	text := update.Message.Text
	id := update.Message.Chat.ID

	if text == Start {
		return bot.start(id, "Что отправить?")
	} else if text == Random {
		return bot.sendRandom(id)
	} else if text == Search {
		return bot.search(id)
	} else if text == Saved {
		//return bot.saved()
	}
	return bot.start(id, "Ошибочка, давай заново!")
}

func (bot *Bot) start(id int, greeting string) error {
	bot.LastQuote = bash.Quote{}
	bot.LastSearchIndex = 0
	bot.NextProcessor = nil
	buttons := newReplyKeyboardMarkup([][]string{
		{Random},
		{Search},
		{Saved},
	})

	_, err := bot.API.sendTextWithKeybord(id, greeting, buttons)
	return err
}

func (bot *Bot) sendRandom(id int) error {
	quotes, err := bash.GetQuotes("random")
	if err != nil {
		return err
	}

	buttons := newReplyKeyboardMarkup([][]string{
		{Other},
		{Plus, Minus, Bayan},
		{Back},
	})

	bot.LastQuote = quotes[rand.Intn(len(quotes))]
	bot.NextProcessor = bot.feedbackQuote

	_, err = bot.API.sendTextWithKeybord(id, bash.QuoteToString(bot.LastQuote), buttons)

	return err
}

func (bot *Bot) feedbackQuote(update *Update) error {
	bot.NextProcessor = nil

	if update.Message == nil {
		return ErrAPINoMessage
	}

	text := update.Message.Text
	id := update.Message.Chat.ID

	if text == Other {
		return bot.sendRandom(id)
	} else if text == Plus {
		go bash.Plus(bot.LastQuote.ID)
		return bot.sendRandom(id)
	} else if text == Minus {
		go bash.Minus(bot.LastQuote.ID)
		return bot.sendRandom(id)
	} else if text == Bayan {
		go bash.Bayan(bot.LastQuote.ID)
		return bot.sendRandom(id)
	} else if text == Back {
		return bot.start(id, "Что отправить?")
	}
	return bot.start(id, "Фигню пишешь!")
}

func (bot *Bot) search(id int) error {

	bot.NextProcessor = bot.search2
	_, err := bot.API.sendTextWithoutKeybord(id, "Введите, что хотите")
	return err

}

func (bot *Bot) search2(update *Update) error {
	bot.NextProcessor = nil

	if update.Message == nil {
		return ErrAPINoMessage
	}

	text := update.Message.Text
	id := update.Message.Chat.ID
	quotes, err := bash.Search(text)
	if err != nil {
		return err
	}

	buttons := newReplyKeyboardMarkup([][]string{
		{Other},
		{Plus, Minus, Bayan},
		{Back},
	})

	bot.LastQuote = quotes[rand.Intn(len(quotes))]
	bot.NextProcessor = bot.feedbackSearch
	bot.SearchRequest = text
	bot.LastSearchIndex++

	_, err = bot.API.sendTextWithKeybord(id, bash.QuoteToString(bot.LastQuote), buttons)
	return err
}

func (bot *Bot) searchLast(id int) error {
	quotes, err := bash.Search(bot.SearchRequest)
	if err != nil {
		return err
	}

	buttons := newReplyKeyboardMarkup([][]string{
		{Other},
		{Plus, Minus, Bayan},
		{Back},
	})

	if bot.LastSearchIndex < len(quotes) {
		bot.LastQuote = quotes[bot.LastSearchIndex]
		bot.NextProcessor = bot.feedbackSearch
		bot.LastSearchIndex++

		_, err = bot.API.sendTextWithKeybord(id, bash.QuoteToString(bot.LastQuote), buttons)
		return err
	}
	bot.LastSearchIndex = 0
	bot.NextProcessor = nil
	return bot.start(id, "Нет больше 8(")
}

func (bot *Bot) feedbackSearch(update *Update) error {
	bot.NextProcessor = nil

	if update.Message == nil {
		return ErrAPINoMessage
	}

	text := update.Message.Text
	id := update.Message.Chat.ID

	if text == Other {
		return bot.searchLast(id)
	} else if text == Plus {
		go bash.Plus(bot.LastQuote.ID)
		return bot.searchLast(id)
	} else if text == Minus {
		go bash.Minus(bot.LastQuote.ID)
		return bot.searchLast(id)
	} else if text == Bayan {
		go bash.Bayan(bot.LastQuote.ID)
		return bot.searchLast(id)
	} else if text == Back {
		return bot.start(id, "Что отправить?")
	}
	return bot.start(id, "Фигню пишешь!")
}
