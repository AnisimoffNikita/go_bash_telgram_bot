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
)

// Bot struct
type Bot struct {
	Token   string
	Self    User
	Client  *http.Client
	Pool    Pool
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
		Token:  token,
		Client: &http.Client{},
		Pool: Pool{
			concurrency: poolSize,
		},
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

	var config Config
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		log.Fatal(err)
	}

	bot, err := newBot(config.Token, config.TimeOut, config.PoolSize)
	if err != nil {
		log.Fatal(err)
	}

	webhookConfig, err := NewWebhookConfig(config.Host,
		config.Port,
		config.Token,
		config.Cert,
		config.PoolSize)

	if err != nil {
		log.Fatal(err)
	}

	_, err = bot.setWebhook(webhookConfig)
	if err != nil {
		log.Fatal(err)
	}

	bot.log("webhook done")

	http.HandleFunc("/"+bot.Token, bot.updateHandler)
	return http.ListenAndServeTLS("0.0.0.0:"+config.Port,
		config.Cert,
		config.PKey,
		nil)
}
func (bot *Bot) updateHandler(w http.ResponseWriter, r *http.Request) {
	_, err := bot.Pool.AddTaskSyncTimed(func() interface{} {
		bytes, _ := ioutil.ReadAll(r.Body)
		fmt.Println(string(bytes))
		return nil
	}, bot.TimeOut)

	if err != nil {
		http.Error(w, fmt.Sprintf("error: %s!\n", err), 500)
	}
}

func (bot *Bot) setWebhook(webhookConfig WebhookConfig) (APIResponse, error) {
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
	endpoint := fmt.Sprintf(APIEndpoint, bot.Token, method)

	resp, err := bot.Client.PostForm(endpoint, params)
	if err != nil {
		return APIResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return APIResponse{}, ErrAPIForbidden
	}

	if resp.StatusCode != http.StatusOK {
		return APIResponse{}, errors.New(http.StatusText(resp.StatusCode))
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

	endpoint := fmt.Sprintf(APIEndpoint, bot.Token, method)

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
