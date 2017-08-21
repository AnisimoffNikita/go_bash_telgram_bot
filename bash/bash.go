package bash

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
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
func GetQuotes(topic string) ([]string, error) {
	address := fmt.Sprintf("http://bash.im/%s", topic)

	res, err := http.Get(address)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	utf8, err := charset.NewReader(res.Body, res.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(utf8)
	if err != nil {
		return nil, err
	}

	node, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	nodes := getElementByClass(node, "text")

	result := make([]string, len(nodes))
	for i, v := range nodes {
		result[i] = ""
		for c := v.FirstChild; c != nil; c = c.NextSibling {
			if c.Data == "br" {
				result[i] += "\n"
			} else {
				result[i] += c.Data
			}
		}
	}

	return result, nil

}
