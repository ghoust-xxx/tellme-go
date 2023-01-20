package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"log"
	"time"
)

const forvoURL = "https://forvo.com"
const audioURL = "https://audio00.forvo.com/audios"
const getRepeats = 10

// mainLoop is process all input word by word
func mainLoop(cfg Config, args []string) {
	if len(args) > 2 {
		log.Fatal("too many words")
	}

	if len(args) == 2 {
		processOneWord(cfg, args[1])
	}
}

// processOneWord try to get pronunciation for a just one word. Useful if we
// have batch reading from a file
func processOneWord(cfg Config, word string) {
	fmt.Printf("%s: word processing...\n", word)
	getPronList(cfg, word)
}

// getPronList gets a pronunciation list for a specific word
func getPronList(cfg Config, word string) {
	fmt.Printf("%s: get pronunciation list...\n", word)

	pageURL := fmt.Sprintf("%s/word/%s/#%s", forvoURL, word, cfg["LANG"])
	fmt.Println(pageURL)
	text, _ := getURL(pageURL)
	fmt.Println(text)
}

// getURL gets a web page, handles possible errors and returns the web page
// content as a string
func getURL(url string) (string, error) {
	client := http.Client {
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(url)
	repeat := getRepeats
	for err != nil && repeat > 0 {
		resp, err = http.Get(url)
		repeat--
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New(fmt.Sprintf("can not get %s: %v", url, err))
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return string(bytes), nil
}
