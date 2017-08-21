package bash

import (
	"golang.org/x/net/html"
)

func getAttribute(n *html.Node, key string) (string, bool) {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val, true
		}
	}
	return "", false
}

func checkClass(n *html.Node, class string) bool {
	if n.Type == html.ElementNode {
		s, ok := getAttribute(n, "class")
		if ok && s == class {
			return true
		}
	}
	return false
}

func traverseClass(n *html.Node, class string) []*html.Node {
	result := make([]*html.Node, 0, 1)
	if checkClass(n, class) {
		result = append(result, n)
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result = append(result, traverseClass(c, class)...)
	}

	return result
}

func getElementByClass(n *html.Node, class string) []*html.Node {
	return traverseClass(n, class)
}
