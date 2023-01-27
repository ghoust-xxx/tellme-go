package main

import (
	"bufio"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

const forvoURL = "https://forvo.com"
const audioURL = "https://audio00.forvo.com/audios"
const testFiles = "local_files"
const getRepeats = 10
const getTimeout = 5 * time.Second
const downloadRepeats = 10
const downloadTimeout = 5 * time.Second

type Pron struct {
	word, author, sex, country, mp3, ogg, aFile, aURL, fullAuthor, cacheDir,
	cacheFile string
}

var getHTML func(url string) (string, error)
var getAudio func(url, dst string) error
var clearScreen func()
var words []string
var tmpDir string

// mainLoop is process all input word by word
func mainLoop(cfg Config, args []string) {
	getHTML = getTestURL
	getAudio = downloadTestFile
	clearScreen = clearScreenInit()

	tmpDir, err := ioutil.TempDir("", "tellme")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	var i int
	var next func() (string, error)
	if len(args) > 0 {
		words = append(words, args...)
		next = func() (string, error) {
			i++
			if i >= len(words) {
				return "", errors.New("out of range")
			}
			return words[i], nil
		}
	} else {
		var f *os.File
		if cfg["FILE"] != "" {
			f, err = os.Open(cfg["FILE"])
		} else {
			f = os.Stdin
		}
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		next = func() (string, error) {
			i++
			if !scanner.Scan() {
				if err = scanner.Err(); err != nil {
					log.Fatal(err)
				}
				return "", errors.New("out of range")
			}
			return scanner.Text(), nil
		}
	}

	i = -1
LOOP:
	for {
		word, err := next()
		if err != nil {
			break
		}

		list := getPronList(cfg, word)

		if cfg["INTERACTIVE"] == "no" {
			if len(list) == 0 {
				continue
			}

			saveWord(list[0])
			goto LOOP
		} else {
			printMenu(i, word, list)
		}
	}
}

// printMenu prints user menu and returs his response (command or a number)
func printMenu(curItem int, word string, list []Pron) string {
UPDATE_PRINT:
	clearScreen()
	fmt.Println(word)
	fmt.Println(strings.Repeat("=", len(word)), "\n")
	digitsNum := len(strconv.Itoa(len(list)))
	for i, item := range list {
		star := " "
		if i == curItem {
			star = "*"
		}
		fmt.Printf("%s %0"+strconv.Itoa(digitsNum)+"d\tBy %s\n",
			star, i, item.fullAuthor)
	}
	fmt.Print("\n\n")
	fmt.Printf("[0-%d]:choose pronunciation    [n]:next word    "+
		"[p]:previous word    [q]:quit\n", len(list)-1)

	allowedChars := "1234567890npq\n"
	var choosenItem string
	for {
		char := getChar()
		if strings.Index(allowedChars, char) == -1 {
			continue
		}
		_, err := strconv.Atoi(char)
		if err == nil {
			choosenItem += char
			fmt.Print(char)
			num, _ := strconv.Atoi(choosenItem)
			if num > len(list)-1 {
				fmt.Println("\nNumber you entered is too big. Press any key...")
				getChar()
				goto UPDATE_PRINT
			}
			if len(choosenItem) == digitsNum {
				return choosenItem
			}
			continue
		}
		return char
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

	return string(char)
}

// saveWord saves mp3/ogg file in cache and in current directory. If cache
// enabled and file already in it returns the word from the cache
func saveWord(item Pron) string {
	fmt.Println("Save word: ", item.word)

	if cfg["CACHE"] == "yes" {
		_, err := os.Stat(item.cacheFile)
		if errors.Is(err, os.ErrNotExist) {
			err = getAudio(item.aURL, item.cacheFile)
			if err != nil {
				return ""
			}
		}

		if cfg["DOWNLOAD"] == "yes" {
			copyFile(item.cacheFile, item.aFile)
			return item.aFile
		}
		return item.cacheFile
	}

	// we do not use cache
	if cfg["DOWNLOAD"] == "yes" {
		err := getAudio(item.aURL, item.aFile)
		if err != nil {
			return ""
		}
	}

	// We have no cache and do not save file in local directory.
	// So we use temporary file if we are in interactive mode.
	// Otherwise we just do not need download anything
	if cfg["INTERACTIVE"] == "yes" {
		file := filepath.Join(tmpDir + filepath.Base(item.cacheFile))
		err := getAudio(item.aURL, file)
		if err != nil {
			return ""
		}
		return file
	}

	return ""
}

// getPronList gets a pronunciation list for a specific word
func getPronList(cfg Config, word string) (result []Pron) {
	pageURL := fmt.Sprintf("%s/word/%s/#%s", forvoURL, word, cfg["LANG"])
	pageText, err := getHTML(pageURL)
	if err != nil {
		log.Printf("can not get pronunciation page for '%s'!\n", word)
		return
	}

	// extract main block with pronunciations
	wordsBlockStr := `(?is)<div id="language-container-` + cfg["LANG"] +
		`.*?<ul.*?>(.*?)</ul>.*?</article>`
	wordsBlockRe := regexp.MustCompile(wordsBlockStr)
	wordsBlock := wordsBlockRe.FindString(pageText)
	if wordsBlock == "" {
		log.Fatal("can not extract words block")
	}

	// extract every pronunciation <li> chunks
	pronStr := `(?is)<li.*?>(.*?)</li>`
	pronRe := regexp.MustCompile(pronStr)
	pronBlocks := pronRe.FindAllString(wordsBlock, -1)
	if pronBlocks == nil {
		log.Fatal("can not extract separate pronunciations blocks")
	}
	for _, chunk := range pronBlocks {
		result = append(result, extractItem(cfg, word, chunk))
	}

	return
}

// extractItem extracts all needed data from one <li> tag
func extractItem(cfg Config, word, chunk string) Pron {
	var item Pron
	item.word = word

	chunkStr := `(?is)onclick="Play\(\d+,.*?,.*?,'(.*?)'.*?>\s*` +
		`Pronunciation by\s+(.*?)\s+` +
		`</span>\s*<span class="from">\((.*?)\ from\ (.*?)\)</span>`
	chunkRe := regexp.MustCompile(chunkStr)
	items := chunkRe.FindStringSubmatch(chunk)
	if items == nil {
		log.Fatal("can not extract items from pronunciation block")
	}

	encodedMp3 := items[1]
	decodedMp3, err := base64.StdEncoding.DecodeString(encodedMp3)
	if err != nil {
		log.Print(err)
	}
	item.mp3 = audioURL + "/mp3/" + string(decodedMp3)
	item.ogg = audioURL + "/ogg/" + string(decodedMp3)
	item.ogg = item.ogg[:len(item.ogg)-3] + "ogg"

	switch cfg["ATYPE"] {
	case "mp3":
		item.aURL = item.mp3
	case "ogg":
		item.aURL = item.ogg
	}

	item.author = items[2]
	authorRe := regexp.MustCompile(`(?si)^<span\ class="ofLink".*?>(.*?)</span>`)
	cleanedAuthor := authorRe.FindStringSubmatch(item.author)
	if len(cleanedAuthor) > 0 {
		item.author = cleanedAuthor[1]
	}

	item.sex = strings.ToLower(items[3])
	item.country = items[4]

	item.fullAuthor = fmt.Sprintf("%s (%s from %s)",
		item.author, item.sex, item.country)

	hash := fmt.Sprintf("%x", md5.Sum([]byte(word)))[0:2]
	item.cacheDir = filepath.Join(cfg["CACHE_DIR"], cfg["ATYPE"],
		cfg["LANG"], hash)

	item.cacheFile = filepath.Join(item.cacheDir,
		word+"_"+item.author+"."+cfg["ATYPE"])

	item.aFile = word + "." + cfg["ATYPE"]

	//printPron(item)
	return item
}

func printPron(p Pron) {
	fmt.Println()
	fmt.Printf("author -> %s\n", p.author)
	fmt.Printf("full author -> %s\n", p.fullAuthor)
	fmt.Printf("sex -> %s\n", p.sex)
	fmt.Printf("country -> %s\n", p.country)
	fmt.Printf("mp3 -> %s\n", p.mp3)
	fmt.Printf("ogg -> %s\n", p.ogg)
	fmt.Printf("cache dir -> %s\n", p.cacheDir)
	fmt.Printf("cache file -> %s\n", p.cacheFile)
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
