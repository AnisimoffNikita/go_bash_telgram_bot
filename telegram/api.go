package telegram

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

// API endpoint mask
const (
	TelegramEndpoint = "https://api.telegram.org/bot%s/%s"
)

// Errors
var (
	ErrAPIKeybord   = errors.New("keybord setting error")
	ErrAPINoMessage = errors.New("not message")
	ErrAPINotOk     = errors.New("not ok")
	ErrAPIForbidden = errors.New("forbidden")
	ErrJobTimedOut  = errors.New("job request timed out")
)

// BotAPI struct
type BotAPI struct {
	Token  string
	Self   User
	Client *http.Client
}

// MakeRequest makes arbitary query
func (bot *BotAPI) MakeRequest(method string, params url.Values) (APIResponse, error) {
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

func (bot *BotAPI) makeMessageRequest(endpoint string, params url.Values) (Message, error) {
	resp, err := bot.MakeRequest(endpoint, params)
	if err != nil {
		return Message{}, err
	}

	var message Message
	json.Unmarshal(resp.Result, &message)

	return message, nil
}

// UploadFile Method
func (bot *BotAPI) UploadFile(method string, params map[string]string, param string, path string) (APIResponse, error) {

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

func (bot *BotAPI) uploadFileRequest(method string, params map[string]string, param string, path string) (*http.Request, error) {
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

// GetMe method
func (bot *BotAPI) GetMe() (User, error) {
	resp, err := bot.MakeRequest("getMe", nil)
	if err != nil {
		return User{}, err
	}

	var user User
	if err := json.Unmarshal(resp.Result, &user); err != nil {
		return User{}, err
	}

	return user, nil
}

// SendText Method
func (bot *BotAPI) SendText(chatID int, text string) (Message, error) {
	params := url.Values{}
	params.Add("chat_id", strconv.Itoa(chatID))
	params.Add("text", text)

	message, err := bot.makeMessageRequest("sendMessage", params)

	if err != nil {
		return Message{}, err
	}

	return message, nil
}

// SendTextWithKeybord Method
func (bot *BotAPI) SendTextWithKeybord(chatID int, text string, keybord ReplyKeyboardMarkup) (Message, error) {
	params := url.Values{}
	params.Add("chat_id", strconv.Itoa(chatID))
	params.Add("text", text)

	keybordJSON, err := json.Marshal(keybord)
	if err != nil {
		return Message{}, ErrAPIKeybord
	}

	params.Add("reply_markup", string(keybordJSON))

	message, err := bot.makeMessageRequest("sendMessage", params)

	if err != nil {
		return Message{}, err
	}

	return message, nil
}

// SendTextWithoutKeybord Method
func (bot *BotAPI) SendTextWithoutKeybord(chatID int, text string) (Message, error) {
	params := url.Values{}
	params.Add("chat_id", strconv.Itoa(chatID))
	params.Add("text", text)

	buttons := ReplyKeyboardRemove{
		RemoveKeyboard: true,
		Selective:      true,
	}

	keybordJSON, err := json.Marshal(buttons)
	if err != nil {
		return Message{}, ErrAPIKeybord
	}

	params.Add("reply_markup", string(keybordJSON))

	message, err := bot.makeMessageRequest("sendMessage", params)

	if err != nil {
		return Message{}, err
	}

	return message, nil
}
