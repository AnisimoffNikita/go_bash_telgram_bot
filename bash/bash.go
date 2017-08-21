package bash

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

// BashEndpoint
const (
	BashEndpoint = "http://bash.im/%s"
)

// GetQuotes comment... wtf
func GetQuotes(topic string) ([]string, error) {
	address := fmt.Sprintf(BashEndpoint, topic)

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
