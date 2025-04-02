package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

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
	body, err := getHTML(os.Args[1])
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(body)
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
