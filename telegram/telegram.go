package telegram

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

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

func (bot *Bot) log(l string) {
	if bot.Debug {
		log.Println(l)
	}
}

func newBot(token string, timeout time.Duration, poolSize int) (*Bot, error) {
	bot := &Bot{
		Token:   token,
		Client:  &http.Client{},
		Pool:    NewPool(poolSize),
		TimeOut: timeout * time.Millisecond,
	}

	bot.Debug = true

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
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("config correct...")

	bot, err := newBot(config.Token, config.TimeOut, config.PoolSize)
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

		_, err := bot.Pool.AddTaskSyncTimed(func() interface{} {
			return bot.processUpdate(update)
		}, bot.TimeOut)

		if err != nil {
			log.Println(err)
		}

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

func (bot *Bot) processUpdate(update *Update) error {
	if update.Message == nil {
		return ErrAPINoMessage
	}

	bot.log(fmt.Sprintf("[%s] %s", update.Message.From.UserName, update.Message.Text))

	quotes, err := bash.GetQuotes(update.Message.Text)
	if err != nil {
		return err
	}

	_, err = bot.sendText(update.Message.Chat.ID, quotes[0])
	return err
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

func (bot *Bot) makeRequest(method string, params url.Values) (APIResponse, error) {
	endpoint := fmt.Sprintf(TelegramEndpoint, bot.Token, method)

	resp, err := bot.Client.PostForm(endpoint, params)
	if err != nil {
		return APIResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return APIResponse{}, ErrAPIForbidden
	}

	if resp.StatusCode != http.StatusOK {
		return APIResponse{}, ErrAPINotOk
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return APIResponse{}, err
	}

	var apiResp APIResponse
	json.Unmarshal(bytes, &apiResp)

	if !apiResp.Ok {
		return apiResp, errors.New(apiResp.Description)
	}

	return apiResp, nil
}

func (bot *Bot) makeMessageRequest(endpoint string, params url.Values) (Message, error) {
	resp, err := bot.makeRequest(endpoint, params)
	if err != nil {
		return Message{}, err
	}

	var message Message
	json.Unmarshal(resp.Result, &message)

	return message, nil
}

func (bot *Bot) uploadFile(method string, params map[string]string, param string, path string) (APIResponse, error) {

	req, err := bot.uploadFileRequest(method, params, param, path)
	if err != nil {
		return APIResponse{}, err
	}

	res, err := bot.Client.Do(req)
	if err != nil {
		return APIResponse{}, err
	}
	defer res.Body.Close()

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return APIResponse{}, err
	}

	var apiResp APIResponse

	if err := json.Unmarshal(bytes, &apiResp); err != nil {
		return APIResponse{}, err
	}

	if !apiResp.Ok {
		return APIResponse{}, errors.New(apiResp.Description)
	}

	return apiResp, nil
}

func (bot *Bot) uploadFileRequest(method string, params map[string]string, param string, path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(param, filepath.Base(path))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf(TelegramEndpoint, bot.Token, method)

	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req, nil
}

func (bot *Bot) getMe() (User, error) {
	resp, err := bot.makeRequest("getMe", nil)
	if err != nil {
		return User{}, err
	}

	var user User
	if err := json.Unmarshal(resp.Result, &user); err != nil {
		return User{}, err
	}

	return user, nil
}

func (bot *Bot) sendText(chatID int64, text string) (Message, error) {
	params := url.Values{}
	params.Add("chat_id", strconv.FormatInt(chatID, 10))
	params.Add("text", text)

	message, err := bot.makeMessageRequest("sendMessage", params)

	if err != nil {
		return Message{}, err
	}

	return message, nil
}
