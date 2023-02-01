package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/term"
)

const getRepeats = 10
const getTimeout = 5 * time.Second
const downloadRepeats = 10
const downloadTimeout = 5 * time.Second
const testFiles = "local_files"

// sayWord tries to play audiofile with pronunciation using mpg123 cmd-line app
func sayWord(path string) {
	mpg123 := exec.Command("mpg123", "-q", path)
	if err := mpg123.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// clearScreenInit prepare platform independent function for terminal clearing
func clearScreen() {
	switch runtime.GOOS {
	case "linux":
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	case "darwin":
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	default:
		return
	}
}

// getChar reads from STDIN one character
func getChar() string {
	state, err := term.MakeRaw(0)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() {
		err := term.Restore(0, state)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}()

	in := bufio.NewReader(os.Stdin)
	char, _, err := in.ReadRune()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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
		fmt.Fprintln(os.Stderr, err)
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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	f, err := os.Create(dst)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return nil
}

// downloadTestFile can be used in tests and download audio file from file system
func downloadTestFile(url, dst string) error {
	dir := filepath.Dir(dst)
	err := os.MkdirAll(dir, 0750)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
