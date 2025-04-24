package main

import (
	"fmt"
	"net/url"
)

func normalizeURL(urlString string) (string, error) {
	url, err := url.Parse(urlString)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%s", url.Host, url.Path), nil
}
