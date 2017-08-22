package telegram

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	yaml "gopkg.in/yaml.v2"

	"../bash"
)

// Bot struct
type Bot struct {
	Token   string
	Self    User
	Client  *http.Client
	Pool    *Pool
	TimeOut time.Duration
	Debug   bool
}

func newBot(token string, timeout time.Duration, poolSize int, debug bool) (*Bot, error) {
	bot := &Bot{
		Token:   token,
		Client:  &http.Client{},
		Pool:    NewPool(poolSize),
		TimeOut: timeout * time.Millisecond,
	}

	bot.Debug = debug

	bot.Pool.Run()

	self, err := bot.getMe()
	if err != nil {
		return nil, err
	}

	bot.Self = self
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
		return bot.processUpdatesChannel()
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

	http.HandleFunc("/"+bot.Token, bot.updateHandler)
	return http.ListenAndServeTLS("0.0.0.0:"+config.Port,
		config.Cert,
		config.PKey,
		nil)
}

func (bot *Bot) processUpdatesChannel() error {
	updates, err := bot.getUpdatesChannel(bot.Pool.concurrency)
	if err != nil {
		return err
	}

	for update := range updates {
		go func(update *Update) {
			_, err := bot.Pool.AddTaskSyncTimed(func() interface{} {
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

			resp, err := bot.makeRequest("getUpdates", params)
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

	resp, err := bot.uploadFile("setWebhook", params, "certificate", webhookConfig.Cert)

	if err != nil {
		return APIResponse{}, err
	}

	return resp, nil
}

func (bot *Bot) processUpdate(update *Update) error {
	if update.Message == nil {
		return ErrAPINoMessage
	}

	text := update.Message.Text

	theme, ok := Themes[text]
	if ok {
		bot.sendQuote(update.Message.Chat.ID, theme)
	}

	if text == Settings {
		bot.sendSettings(update.Message.Chat.ID)
	}

	_, err := bot.sendText(update.Message.Chat.ID, "ошибочка, сорян")
	return err
}

func (bot *Bot) sendQuote(id int, theme string) error {
	quotes, err := bash.GetQuotes(theme)
	if err != nil {
		return err
	}

	buttons := newReplyKeyboardMarkup([][]string{
		{Random, New},
		{ByRating, Best},
		{Abyss, AbyssTop},
		{AbyssBest, Settings},
	})

	_, err = bot.sendTextWithKeybord(id, quotes[0], buttons)
	return err
}

func (bot *Bot) sendSettings(id int) error {
	buttons := newReplyKeyboardMarkup([][]string{
		{Random, New},
		{ByRating, Best},
		{Abyss, AbyssTop},
		{AbyssBest, Settings},
	})

	_, err := bot.sendTextWithKeybord(id, "Скоро тут будут настройки...", buttons)
	return err
}
