package bash

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// Quote comment... wtf
type Quote struct {
	Site string `json:"site"`
	Name string `json:"name"`
	Desc string `json:"desc"`
	Link string `json:"link"`
	Text string `json:"elementPureHtml"`
}

// GetQuotes comment... wtf
func GetQuotes(topic string, n int) ([]Quote, error) {
	data, err := readData(topic, n)
	if err != nil {
		return nil, err
	}

	var quotes []Quote
	if err := json.Unmarshal(data, &quotes); err != nil {
		return nil, err
	}

	processText(&quotes)

	return quotes, nil
}

func readData(topic string, n int) ([]byte, error) {
	address := fmt.Sprintf("http://umorili.herokuapp.com/api/get?site=bash.im&name=%s&num=%d", topic, n)

	res, err := http.Get(address)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	fmt.Println(string(data))

	return data, nil
}

func processText(quotes *[]Quote) {
	replacer := strings.NewReplacer("<p>", "",
		"</p>", "",
		"<br />", "",
		"&nbsp;", " ",
		"&raquo;", "»",
		"&laquo;", "«")
	for i := range *quotes {
		(*quotes)[i].Text = replacer.Replace((*quotes)[i].Text)
	}
}
