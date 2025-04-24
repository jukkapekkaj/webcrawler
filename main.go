package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
)

type config struct {
	pages              map[string]int
	baseURL            *url.URL
	mu                 *sync.Mutex
	concurrencyControl chan struct{}
	wg                 *sync.WaitGroup
	maxPages           int
}

func main() {
	args := os.Args[1:]
	if len(args) < 3 {
		fmt.Println("no website provided")
		os.Exit(1)
	}
	if len(args) > 3 {
		fmt.Println("too many arguments provided")
		os.Exit(1)
	}
	base_url := args[0]
	fmt.Printf("starting crawl of: %s\n", base_url)

	parsed_url, err := url.Parse(base_url)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	concurrent, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if concurrent < 1 {
		concurrent = 1
	}

	maxPages, err := strconv.Atoi(args[2])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	c := &config{
		pages:              make(map[string]int),
		baseURL:            parsed_url,
		mu:                 &sync.Mutex{},
		concurrencyControl: make(chan struct{}, concurrent),
		wg:                 &sync.WaitGroup{},
		maxPages:           maxPages,
	}

	c.wg.Add(1)
	go c.crawlPage(base_url)

	fmt.Println("Main waiting...")
	c.wg.Wait()
	fmt.Println("Main exiting...")
	printReport(c.pages, c.baseURL.String())

}

func (c *config) crawlPage(rawCurrentURL string) {
	fmt.Println("Crawling ", rawCurrentURL)
	defer c.wg.Done()
	defer func() {
		<-c.concurrencyControl
	}()
	c.concurrencyControl <- struct{}{}

	c.mu.Lock()
	if len(c.pages) >= c.maxPages {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	urlCurrent, err := url.Parse(rawCurrentURL)
	if err != nil {
		fmt.Println("38:", err)
		return
	}

	//fmt.Println(rawCurrentURL, c.baseURL.Host, urlCurrent.Host)
	if c.baseURL.Host != urlCurrent.Host {
		//fmt.Println("Different domain, return early")
		return
	}

	url, err := normalizeURL(rawCurrentURL)
	if err != nil {
		fmt.Println("49:", err)
		return
	}

	if !c.addPageVisit(url) {
		//fmt.Println("Page already visited:", url)
		return
	}

	content, err := getHTML(rawCurrentURL)
	if err != nil {
		fmt.Println("63:", err)
		return
	}
	//fmt.Println("Got HTML from ", rawCurrentURL)
	urls, err := getURLsFromHTLM(content, c.baseURL.String())
	if err != nil {
		fmt.Println("69:", err)
		return
	}
	//fmt.Println("New URLs:", urls)

	for _, u := range urls {
		c.wg.Add(1)
		go c.crawlPage(u)
	}

}

type hits struct {
	url   string
	count int
}

func printReport(pages map[string]int, baseURL string) {
	fmt.Printf(`=============================
  REPORT for %s
=============================
`, baseURL)

	s := make([]hits, 0, len(pages))
	for key, val := range pages {
		s = append(s, hits{url: key, count: val})
	}

	slices.SortFunc(s, func(a, b hits) int {
		if a.count < b.count {
			return 1
		} else if a.count > b.count {
			return -1
		} else {
			if a.url < b.url {
				return 1
			} else if a.url > b.url {
				return -1
			} else {
				return 0
			}
		}
	})

	for _, item := range s {
		fmt.Printf("Found %d internal links to %s\n", item.count, item.url)
	}

}

func (cfg *config) addPageVisit(normalizedURL string) (isFirst bool) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	_, ok := cfg.pages[normalizedURL]
	if ok {
		cfg.pages[normalizedURL] += 1
	} else {
		cfg.pages[normalizedURL] = 1
	}
	return !ok
}

func getHTML(rawURL string) (string, error) {
	fmt.Println("GET:", rawURL)
	res, err := http.Get(rawURL)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return "", fmt.Errorf("invalid status code: %d", res.StatusCode)
	}

	if !strings.Contains(res.Header.Get("Content-Type"), "text/html") {
		return "", fmt.Errorf("invalid Content-Type: %s", res.Header.Get("Content-Type"))
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil

}
