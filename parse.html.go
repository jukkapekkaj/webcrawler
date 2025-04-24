package main

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

func getURLsFromHTLM(htmlBody, rawBaseUrl string) ([]string, error) {
	reader := strings.NewReader(htmlBody)
	root, err := html.Parse(reader)
	if err != nil {
		return nil, err
	}
	links := make([]string, 0)
	for node := range root.Descendants() {
		if node.Type == html.ElementNode && node.Data == "a" {
			for _, attr := range node.Attr {
				if attr.Key == "href" {
					if strings.HasPrefix(attr.Val, "/") {
						links = append(links, fmt.Sprintf("%s%s", rawBaseUrl, attr.Val))
					} else {
						links = append(links, attr.Val)
					}
					break
				}
			}
		}
	}
	return links, nil
}
