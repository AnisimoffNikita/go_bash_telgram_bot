package bash

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// Quote struct
type Quote struct {
	Text   string
	Rating string
	ID     string
}

// GetQuotes comment... wtf
func GetQuotes(topic string) ([]Quote, error) {
	address := fmt.Sprintf("http://bash.im/%s", topic)

	res, err := http.Get(address)
	if err != nil {
		return nil, fmt.Errorf("can't get page: %s", err)
	}
	defer res.Body.Close()

	utf8, err := charset.NewReader(res.Body, res.Header.Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("GetQuotes: %s", err)
	}

	data, err := ioutil.ReadAll(utf8)
	if err != nil {
		return nil, fmt.Errorf("GetQuotes: %s", err)
	}

	node, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("GetQuotes: %s", err)
	}

	return getQuotesFromHTML(node)
}

// GetQuoteByID gets quote by id
func GetQuoteByID(id string) (Quote, error) {
	address := fmt.Sprintf("http://bash.im/quote/%s", id)

	res, err := http.Get(address)
	if err != nil {
		return Quote{}, err
	}
	defer res.Body.Close()

	utf8, err := charset.NewReader(res.Body, res.Header.Get("Content-Type"))
	if err != nil {
		return Quote{}, err
	}

	data, err := ioutil.ReadAll(utf8)
	if err != nil {
		return Quote{}, err
	}

	node, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return Quote{}, err
	}

	quotes, err := getQuotesFromHTML(node)
	if err != nil {
		return Quote{}, err
	}
	if len(quotes) == 0 {
		return Quote{}, errors.New("no quote")
	}
	return quotes[0], nil
}

// QuoteToString convers Quote to String
func QuoteToString(quote Quote) string {
	str := ""

	str += quote.Text + "\n\n"
	str += "# " + quote.ID + "\n"
	str += "+ " + quote.Rating + "\n"

	return str
}

// Plus request
func Plus(id string) {
	address := fmt.Sprintf("http://bash.im/quote/%s/rulez", id)
	data := fmt.Sprintf("quote=%s&act=rulez", id)

	client := &http.Client{}
	req, _ := http.NewRequest("POST", address, strings.NewReader(data))
	req.Header.Add("Referer", "http://bash.im/")

	resp, _ := client.Do(req)

	defer resp.Body.Close()
}

// Minus request
func Minus(id string) {
	address := fmt.Sprintf("http://bash.im/quote/%s/sux", id)

	data := fmt.Sprintf("quote=%s&act=sux", id)

	client := &http.Client{}
	req, _ := http.NewRequest("POST", address, strings.NewReader(data))
	req.Header.Add("Referer", "http://bash.im/")
	resp, _ := client.Do(req)
	defer resp.Body.Close()
}

// Bayan request
func Bayan(id string) {
	address := fmt.Sprintf("http://bash.im/quote/%s/bayan", id)

	data := fmt.Sprintf("quote=%s&act=bayan", id)

	client := &http.Client{}
	req, _ := http.NewRequest("POST", address, strings.NewReader(data))
	req.Header.Add("Referer", "http://bash.im/")

	resp, _ := client.Do(req)
	defer resp.Body.Close()
}

// Search func searches on bash
func Search(req string) ([]Quote, error) {
	//url encode Windows1251
	buf := new(bytes.Buffer)
	wToWin1251 := transform.NewWriter(buf, charmap.Windows1251.NewEncoder())
	io.Copy(wToWin1251, strings.NewReader(req))
	wToWin1251.Close()

	parts := make([]string, 0)
	for _, v := range buf.Bytes() {
		parts = append(parts, fmt.Sprintf("%%%X", v))
	}

	reqEncode := strings.Join(parts, "")
	address := fmt.Sprintf("http://bash.im/index?text=%s", reqEncode)

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

	return getQuotesFromHTML(node)
}

func getQuotesFromHTML(node *html.Node) ([]Quote, error) {

	quoteNodes := getElementByClass(node, "quote")
	quotes := make([]Quote, len(quoteNodes))

	for i, q := range quoteNodes {
		idClass := getElementByClass(q, "id")
		if len(idClass) == 0 {
			continue
		}
		idNode := idClass[0]
		ratingClass := getElementByClass(q, "rating")
		if len(ratingClass) == 0 {
			continue
		}
		ratingNode := ratingClass[0]
		textClass := getElementByClass(q, "text")
		if len(textClass) == 0 {
			continue
		}
		textNode := textClass[0]
		quotes[i].ID = idNode.FirstChild.Data[1:]
		quotes[i].Rating = ratingNode.FirstChild.Data
		quotes[i].Text = ""
		for c := textNode.FirstChild; c != nil; c = c.NextSibling {
			if c.Data == "br" {
				quotes[i].Text += "\n"
			} else {
				quotes[i].Text += c.Data
			}
		}
	}
	return quotes, nil
}
