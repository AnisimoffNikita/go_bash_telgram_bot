package bash

import (
	"bytes"
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

// BashEndpoint
const (
	BashEndpoint = "http://bash.im/%s"
)

// Quote struct
type Quote struct {
	Text   string
	Rating string
	ID     string
}

// GetQuotes comment... wtf
func GetQuotes(topic string) ([]Quote, error) {
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

	return getQuotesFromHTML(node)
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
		idNode := getElementByClass(q, "id")[0]
		ratingNode := getElementByClass(q, "rating")[0] // there is only one class
		textNode := getElementByClass(q, "text")[0]     // there is only one class
		quotes[i].ID = idNode.FirstChild.Data[1:]       // # char skip
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
