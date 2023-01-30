package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/term"
)

// sayWord tries to play audiofile with pronunciation using mpg123 cmd-line app
func sayWord(path string) {
	mpg123 := exec.Command("mpg123", "-q", path)
	if err := mpg123.Run(); err != nil {
		log.Fatal(err)
	}
}

// clearScreenInit prepare platform independent function for terminal clearing
func clearScreenInit() func() {
	switch runtime.GOOS {
	case "linux":
		return func() {
			cmd := exec.Command("clear")
			cmd.Stdout = os.Stdout
			cmd.Run()
		}
	case "darwin":
		return func() {
			cmd := exec.Command("clear")
			cmd.Stdout = os.Stdout
			cmd.Run()
		}
	case "windows":
		return func() {
			cmd := exec.Command("cmd", "/c", "cls")
			cmd.Stdout = os.Stdout
			cmd.Run()
		}
	default:
		return func() {}
	}
}

// getChar reads from STDIN one character
func getChar() string {
	state, err := term.MakeRaw(0)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err := term.Restore(0, state)
		if err != nil {
			log.Fatal(err)
		}
	}()

	in := bufio.NewReader(os.Stdin)
	char, _, err := in.ReadRune()
	if err != nil {
		log.Fatal(err)
	}

	if char == '\r' {
		char = '\n'
	}

	return string(char)
}

// getURL gets a web page, handles possible errors and returns the web page
// content as a string
func getURL(url string) (string, error) {
	client := http.Client{
		Timeout: getTimeout,
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

// getTestURL can be used in tests and gets web pages from file system
func getTestURL(url string) (string, error) {
	last := strings.LastIndex(url, "/")
	first := strings.LastIndex(url[:last], "/") + 1
	f, err := os.Open(filepath.Join(
		testFiles, "forvo_"+cfg["LANG"]+"_"+url[first:last]+".html"))
	if err != nil {
		log.Print(err)
	}
	defer f.Close()
	text, _ := io.ReadAll(f)
	pageText := string(text)

	return pageText, err
}

// downloadFile gets and saves audiofile from web. In case of enabled cache it
// first checks cache directory. If file is missing function downloads it
// to the cache directory and then copy it to the current location.
func downloadFile(url, dst string) error {
	fmt.Println("download...")
	dir := filepath.Dir(dst)
	err := os.MkdirAll(dir, 0750)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Create(dst)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	client := http.Client{
		Timeout: getTimeout,
	}
	resp, err := client.Get(url)
	repeat := getRepeats
	for err != nil && repeat > 0 {
		resp, err = http.Get(url)
		repeat--
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("can not download %s: %v", url, err))
	}

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

// downloadTestFile can be used in tests and download audio file from file system
func downloadTestFile(url, dst string) error {
	dir := filepath.Dir(dst)
	err := os.MkdirAll(dir, 0750)
	if err != nil {
		log.Fatal(err)
	}

	first := strings.LastIndex(dst, "/")
	last := first + strings.Index(dst[first:], "_")
	src := filepath.Join(testFiles,
		"forvo_"+cfg["LANG"]+"_"+dst[first+1:last]+"."+cfg["ATYPE"])

	copyFile(src, dst)

	return nil
}

// copyFile just a helper function to copy file in a more comfortable way
func copyFile(src, dst string) {
	in, err := os.Open(src)
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		log.Fatal(err)
	}
}
