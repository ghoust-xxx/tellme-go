package main

import (
	"bufio"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
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
var getWord func(i int) (string, error)
var clearScreen func()
var words []string
var scanner *bufio.Scanner
var tmpDir string
var inputFile *os.File

// mainLoop is process all input word by word
func mainLoop(cfg Config, args []string) {
	getHTML = getTestURL
	getAudio = downloadTestFile
	clearScreen = clearScreenInit()
	wordListInit(cfg, args)

	tmpDir, err := ioutil.TempDir("", "tellme")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	i := 0
	curItem := 0
	var errMessage string
	newWord := true
	var list []Pron
	word, err := getWord(i)
	for {
		if newWord {
			list = getPronList(cfg, word)
			newWord = false
		}

		if cfg["INTERACTIVE"] == "no" {
			if len(list) == 0 {
				continue
			}
			saveWord(list[0])
			continue
		} else {
			key := printMenu(curItem, word, list, errMessage)
			errMessage = ""
			switch key {
			case "q":
				return
			case "p":
				i--
				curItem = 0
				newWord = true
				word, err = getWord(i)
				if err != nil {
					errMessage = err.Error()
					i++
				}
			case "n", "\n":
				i++
				curItem = 0
				newWord = true
				word, err = getWord(i)
				if err != nil {
					errMessage = err.Error()
					i--
				}
			case "r":
			case "j":
				curItem++
				if curItem >= len(list) {
					curItem = 0
				}
			case "k":
				curItem--
				if curItem < 0 {
					curItem = len(list) - 1
				}
			default:
				curItem, _ = strconv.Atoi(key)
			}
		}
	}

	inputFile.Close()
}

// wordListInit populate word list in case of cmd-line arguments or open
// input file/STDIO for reading otherwise
func wordListInit(cfg Config, args []string) {
	if len(args) > 0 {
		words = append(words, args...)
		getWord = getWordArgs
	} else {
		var err error
		if cfg["FILE"] != "" {
			inputFile, err = os.Open(cfg["FILE"])
		} else {
			inputFile = os.Stdin
		}
		if err != nil {
			log.Fatal(err)
		}

		scanner = bufio.NewScanner(inputFile)
		getWord = getWordFile
	}
}

// getWordArgs return i-th word if we used cmd-line arguments
func getWordArgs(i int) (string, error) {
	if i >= 0 && i < len(words) {
		return words[i], nil
	}

	if i < 0 {
		return words[0], errors.New("The beginning of the list")
	}

	if i >= len(words) {
		return words[len(words)-1], errors.New("The end of the list")
	}
	return "", nil
}

// getWordFile return i-th word if we used file or STDIO as words source
func getWordFile(i int) (string, error) {
	if i >= 0 && i < len(words) {
		return words[i], nil
	}

	if i < 0 {
		return words[0], errors.New("The beginning of the list")
	}

	if i >= len(words) {
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				log.Fatal(err)
			}
			return words[len(words)-1], errors.New("The end of the list")
		}
		words = append(words, scanner.Text())
		return words[len(words)-1], nil
	}
	return "", nil
}

// printMenu prints user menu and returs his response (command or a number)
func printMenu(curItem int, word string, list []Pron, errMessage string) string {
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
	fmt.Printf("[0-%d]:choose pronunciation    [n|<Enter>]:next word    [p]:previous word\n"+
		"[j]:next pronunciation    [k]:previous pronunciation    [r]:repeat again    [q]:quit\n",
		len(list)-1)
	if errMessage != "" {
		fmt.Printf("\n%s\n", errMessage)
	}

	aPath := saveWord(list[curItem])
	sayWord(aPath)

	allowedChars := "\n1234567890npjkrq"
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
