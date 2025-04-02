package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

type config struct {
	pages              map[string]int
	baseURL            *url.URL
	mu                 *sync.Mutex
	concurrencyControl chan struct{}
	wg                 *sync.WaitGroup
}

func main() {
	if len(os.Args) > 2 {
		fmt.Println("too many arguments provided")
		os.Exit(1)
	}
	if len(os.Args) < 2 {
		fmt.Println("no website provided")
		os.Exit(1)
	}
	fmt.Printf("starting crawl of: %v\n", os.Args[1])
	pages := map[string]int{}
	baseUrl, err := url.Parse(os.Args[1])
	if err != nil {
		fmt.Println("cant parse URL")
		os.Exit(1)
	}
	control := make(chan struct{}, 1)
	conf := config{
		pages:              pages,
		baseURL:            baseUrl,
		mu:                 &sync.Mutex{},
		concurrencyControl: control,
		wg:                 &sync.WaitGroup{},
	}
	//crawlPage(os.Args[1], os.Args[1], pages)
	conf.crawlPage(baseUrl.String())
}

func getHTML(rawURL string) (string, error) {
	client := http.Client{}
	url, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	req := http.Request{}
	req.Method = "GET"
	req.URL = url
	resp, err := client.Do(&req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", errors.New(fmt.Sprintf("Error, page responsded with %d", resp.StatusCode))
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		return "", errors.New("Content type is not text/html")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil

}

func crawlPage(rawBaseURL, rawCurrentURL string, pages map[string]int) {
	baseURL, err := url.Parse(rawBaseURL)
	if err != nil {
		fmt.Println("Cant parse base url")
		return

	}
	currentURL, err := url.Parse(rawCurrentURL)
	if err != nil {
		fmt.Println("Cant parse current url")
		return
	}
	if currentURL.Host != baseURL.Host {
		fmt.Println(pages)
		return
	}
	normalCurrentURL, err := normalizeURL(rawCurrentURL)
	if err != nil {
		fmt.Println(err)
		return
	}
	val, ok := pages[normalCurrentURL]
	if ok {
		pages[normalCurrentURL] = val + 1
		return
	}
	pages[normalCurrentURL] = 1
	fmt.Println("getting HTML for ", currentURL.String())
	currHTML, err := getHTML(currentURL.String())
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(currHTML)

	urlsInPage, err := getURLsFromHTML(currHTML, currentURL.String())
	for _, v := range urlsInPage {
		crawlPage(rawBaseURL, v, pages)
	}
}

func (cfg *config) crawlPage(rawCurrentURL string) {
	cfg.concurrencyControl <- struct{}{}
	defer func() {
		<-cfg.concurrencyControl
		cfg.wg.Done()
	}()
	currentURL, err := url.Parse(rawCurrentURL)
	if err != nil {
		fmt.Println("Cant parse current url")
		return
	}
	if currentURL.Host != cfg.baseURL.Host {
		fmt.Printf("%v is not in the same domain as %v \n", currentURL, cfg.baseURL.String())
		return
	}
	normalCurrentURL, err := normalizeURL(rawCurrentURL)
	if err != nil {
		fmt.Println(err)
		return
	}
	isFirst := cfg.addPageVisit(normalCurrentURL)
	if !isFirst {
		fmt.Printf("URL $v already visiting, skipping\n", normalCurrentURL)
	}

	fmt.Println("getting HTML for ", currentURL.String())
	currHTML, err := getHTML(currentURL.String())
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(currHTML)

	urlsInPage, err := getURLsFromHTML(currHTML, currentURL.String())
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, nextURL := range urlsInPage {
		cfg.wg.Add(1)
		go cfg.crawlPage(nextURL)
		fmt.Println("Called crawl on ", nextURL)
	}
}

func (cfg *config) addPageVisit(normalizedURL string) (isFirst bool) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	fmt.Println("evaluating ", normalizedURL)

	if _, visited := cfg.pages[normalizedURL]; visited {
		fmt.Printf("%v already exists in map. incrementing \n", normalizedURL)
		cfg.pages[normalizedURL]++
		fmt.Println("map after incrementing", cfg.pages)
		return false
	}
	fmt.Printf("%v doesnt exists in map. adding \n", normalizedURL)
	cfg.pages[normalizedURL] = 1
	fmt.Println("map after adding", cfg.pages)
	return true
}
