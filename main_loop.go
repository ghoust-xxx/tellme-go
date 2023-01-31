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

	if cfg["INTERACTIVE"] == "no" {
		if len(args) > 0 {
			loopNonInArgs(cfg, args)
			return
		}
		if cfg["FILE"] != "" {
			loopNonInFile(cfg)
			return
		}
		loopNonInStdin(cfg)
		return
	} else if cfg["INTERACTIVE"] == "yes" {
		if len(args) > 0 {
			loopInArgs(cfg, args)
			return
		}
		if cfg["FILE"] != "" {
			loopInFile(cfg)
			return
		}
		loopInStdin(cfg)
		return
	}
	return
}

func loopNonInArgs(cfg Config, args []string) {
	for _, word := range args {
		if word == "" {
			continue
		}
		list := getPronList(cfg, word)
		if len(list) > 0 {
			saveWord(list[0])
		}
	}
}

func loopNonInFile(cfg Config) {
	file, err := os.Open(cfg["FILE"])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		word := scanner.Text()
		if word == "" {
			continue
		}
		list := getPronList(cfg, word)
		if len(list) > 0 {
			saveWord(list[0])
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
}

func loopNonInStdin(cfg Config) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		word := scanner.Text()
		if word == "" {
			continue
		}
		list := getPronList(cfg, word)
		if len(list) > 0 {
			saveWord(list[0])
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
}

func loopInArgs(cfg Config, args []string) {
	for i := 0; i < len(args); i++ {
		if args[i] != "" {
			words = append(words, args[i])
		}
	}
	if len(words) == 0 {
		return
	}

	wordIdx := 0
	pronIdx := 0
	for {
		list := getPronList(cfg, words[wordIdx])
		if len(list) == 0 {
			printNoPron(words[wordIdx], wordIdx == 0, wordIdx == len(words)-1)
		}
		key := printMenu(list, pronIdx, wordIdx == 0, wordIdx == len(words)-1)
		switch key {
		case "q":
			return
		case "p":
			wordIdx--
			pronIdx = 0
		case "n", "\n":
			wordIdx++
			pronIdx = 0
		case "r":
		case "j":
			pronIdx++
		case "k":
			pronIdx--
		default:
		}
	}
}

func loopInFile(cfg Config) {
}

func loopInStdin(cfg Config) {
}

func printNoPron(word string, isFirstWord, isLastWord bool) string {
	optLine := fmt.Sprintf("Can not get pronunciation for `%s`\n\n", word)
	allowedChars := "tq"
	if !isLastWord {
		optLine += "[n|<Enter>]:next word    "
		allowedChars += "n\n"
	}
	if !isFirstWord {
		optLine += "[p]:previous word    "
		allowedChars += "p"
	}
	optLine += "[t]:try again    [q]:quit\n"

	clearScreen()
	fmt.Println(word)
	fmt.Println(strings.Repeat("=", len(word)), "\n\n")
	fmt.Print(optLine)

	for {
		char := getChar()
		if strings.Index(allowedChars, char) == -1 {
			continue
		}
		return char
	}
}

func printMenu(list []Pron, pronIdx int, isFirstWord, isLastWord bool) string {
	word := list[0].word
	pronLines := word + "\n"
	pronLines += fmt.Sprintln(strings.Repeat("=", len(word)))
	digitsNum := len(strconv.Itoa(len(list)))
	for i, item := range list {
		star := " "
		if i == pronIdx {
			star = "*"
		}
		pronLines += fmt.Sprintf("%s %0"+strconv.Itoa(digitsNum)+"d\tBy %s\n",
			star, i, item.fullAuthor)
	}
	pronLines += "\n"

	optLine := ""
	allowedChars := "rq"
	if len(list) != 1 {
		optLine += fmt.Sprintf("[0-%d]:choose pronunciation    ", len(list)-1)
		allowedChars += "1234567890"
	}
	if pronIdx != len(list)-1 {
		optLine += "[j]:next pronunciation    "
		allowedChars += "j"
	}
	if pronIdx != 0 {
		optLine += "[k]:previous pronunciation    "
		allowedChars += "k"
	}
	optLine += "[r]:replay sound    "

	if !isLastWord {
		optLine += "[n|<Enter>]:next word    "
		allowedChars += "n\n"
	}
	if !isFirstWord {
		optLine += "[p]:previous word"
		allowedChars += "p"
	}
	optLine += "[q]:quit\n"

	alreadySaid := false
UPDATE_PRINT:
	clearScreen()
	fmt.Print(pronLines)
	fmt.Print(optLine)

	if !alreadySaid {
		aPath := saveWord(list[pronIdx])
		sayWord(aPath)
		alreadySaid = true
	}

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
	if cfg["VERBOSE"] == "yes" {
		fmt.Printf("Saving audio file: `%s`\n", item.aFile)
	}

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
	if cfg["VERBOSE"] == "yes" {
		fmt.Printf("Extracting pronunciation list for `%s`\n", word)
	}
	pageURL := fmt.Sprintf("%s/word/%s/#%s", forvoURL, word, cfg["LANG"])
	pageText, err := getHTML(pageURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can not get pronunciation page for '%s'!\n", word)
		return
	}

	// extract main block with pronunciations
	wordsBlockStr := `(?is)<div id="language-container-` + cfg["LANG"] +
		`.*?<ul.*?>(.*?)</ul>.*?</article>`
	wordsBlockRe := regexp.MustCompile(wordsBlockStr)
	wordsBlock := wordsBlockRe.FindString(pageText)
	if wordsBlock == "" {
		fmt.Fprintln(os.Stderr, "can not extract words block")
		os.Exit(1)
	}

	// extract every pronunciation <li> chunks
	pronStr := `(?is)<li.*?>(.*?)</li>`
	pronRe := regexp.MustCompile(pronStr)
	pronBlocks := pronRe.FindAllString(wordsBlock, -1)
	if pronBlocks == nil {
		fmt.Fprintln(os.Stderr, "can not extract separate pronunciations blocks")
		os.Exit(1)
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
		fmt.Fprintln(os.Stderr, "can not extract items from pronunciation block")
		os.Exit(1)
	}

	encodedMp3 := items[1]
	decodedMp3, err := base64.StdEncoding.DecodeString(encodedMp3)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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

	return item
}
