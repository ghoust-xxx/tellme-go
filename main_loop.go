package main

import (
	"bufio"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const forvoURL = "https://forvo.com"
const audioURL = "https://audio00.forvo.com/audios"

type Pron struct {
	word, author, sex, country, mp3, ogg, aFile, aURL, fullAuthor, cacheDir,
	cacheFile string
}

var getHTML func(cfg Config, url string) (string, error)
var getAudio func(cfg Config, url, dst string) error
var getWord func(i int) (string, error)
var tmpDir string

// mainLoop is process all input word by word
func mainLoop(cfg Config, args []string) {
	getHTML = getURL
	getAudio = downloadFile

	tmpDir, err := ioutil.TempDir("", "tellme")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	if cfg["INTERACTIVE"] == "no" {
		if len(args) > 0 {
			loopNonInArgs(cfg, args)
		} else if cfg["FILE"] != "" {
			loopNonInFile(cfg)
		} else {
			loopNonInStdin(cfg)
		}
	} else if cfg["INTERACTIVE"] == "yes" {
		if len(args) > 0 {
			loopInArgs(cfg, args)
		} else if cfg["FILE"] != "" {
			loopInFile(cfg)
		} else {
			loopInStdin(cfg)
		}
	}
	return
}

// loopNonInArgs is loop for non-interactive processing with getting words from
// argument list
func loopNonInArgs(cfg Config, args []string) {
	for _, word := range args {
		if word == "" {
			continue
		}
		list := getPronList(cfg, word)
		if len(list) > 0 {
			saveWord(cfg, list[0])
		}
	}
}

// loopNonInFile is loop for non-interactive processing with getting words from
// the file
func loopNonInFile(cfg Config) {
	file, err := os.Open(cfg["FILE"])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		word := scanner.Text()
		if word == "" {
			continue
		}
		list := getPronList(cfg, word)
		if len(list) > 0 {
			saveWord(cfg, list[0])
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// loopNonInStdin is loop for non-interactive processing with getting words from
// standart input
func loopNonInStdin(cfg Config) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		word := scanner.Text()
		if word == "" {
			continue
		}
		list := getPronList(cfg, word)
		if len(list) > 0 {
			saveWord(cfg, list[0])
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// loopInArgs is loop for interactive processing with getting words from
// argument list
func loopInArgs(cfg Config, args []string) {
	var words []string
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
	var key string
	for {
		list := getPronList(cfg, words[wordIdx])
		if len(list) == 0 {
			key = printNoPron(words[wordIdx], wordIdx == 0, wordIdx == len(words)-1)
		} else {
			key = printMenu(cfg, list, pronIdx, wordIdx == 0, wordIdx == len(words)-1)
		}
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
		case "t":
		case "e":
			newWord := getNewWord()
			words = append(words[:wordIdx+1], words[wordIdx:]...)
			words[wordIdx] = newWord
			pronIdx = 0
		default:
			pronIdx, _ = strconv.Atoi(key)
		}
	}
}

// loopInFile is loop for interactive processing with getting words from
// the file
func loopInFile(cfg Config) {
	var words []string
	file, err := os.Open(cfg["FILE"])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	wordIdx := 0
	pronIdx := 0
	eof := false
	var key string
	for {
		// read a new word
		if wordIdx == len(words) && !eof {
			if scanner.Scan() {
				newWord := scanner.Text()
				if newWord == "" {
					continue
				}
				words = append(words, newWord)
				wordIdx = len(words) - 1
				pronIdx = 0
			} else if err := scanner.Err(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			} else {
				wordIdx = len(words) - 1
				eof = true
			}
		}
		list := getPronList(cfg, words[wordIdx])
		if len(list) == 0 {
			key = printNoPron(words[wordIdx], wordIdx == 0, wordIdx == len(words)-1 && eof)
		} else {
			key = printMenu(cfg, list, pronIdx, wordIdx == 0, wordIdx == len(words)-1 && eof)
		}
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
		case "t":
		case "e":
			newWord := getNewWord()
			words = append(words[:wordIdx+1], words[wordIdx:]...)
			words[wordIdx] = newWord
			pronIdx = 0
		default:
			pronIdx, _ = strconv.Atoi(key)
		}
	}
}

// loopInStdin is loop for interactive processing with getting words from
// standart input
func loopInStdin(cfg Config) {
	var words []string
	wordIdx := 0
	pronIdx := 0
	var key string

	newWord := getNewWord()
	words = append(words, newWord)
	for {
		list := getPronList(cfg, words[wordIdx])
		if len(list) == 0 {
			key = printNoPron(words[wordIdx], wordIdx == 0, wordIdx == len(words)-1)
		} else {
			key = printMenu(cfg, list, pronIdx, wordIdx == 0, wordIdx == len(words)-1)
		}
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
		case "t":
		case "e":
			newWord := getNewWord()
			words = append(words[:wordIdx+1], words[wordIdx:]...)
			words[wordIdx] = newWord
			pronIdx = 0
		default:
			pronIdx, _ = strconv.Atoi(key)
		}
	}
}

// getNewWord shows promt for user and returns entered word
func getNewWord() string {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Enter a new word: ")
	for scanner.Scan() {
		word := scanner.Text()
		if word == "" {
			fmt.Print("Enter a new word: ")
			continue
		}
		return word
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return ""
}

// printNoPron handles case if we can not find pronunciations for this word.
// Could be be some network issues. In this case you can retray
func printNoPron(word string, isFirstWord, isLastWord bool) string {
	optLine := fmt.Sprintf("Can not get pronunciation for `%s`\n\n", word)
	allowedChars := "tqe"
	if !isLastWord {
		optLine += "[n|<Enter>]:next word    "
		allowedChars += "n\n"
	}
	if !isFirstWord {
		optLine += "[p]:previous word    "
		allowedChars += "p"
	}
	optLine += "[t]:try again    [e]:enter a new word    [q]:quit\n"

	clearScreen()
	fmt.Println(word)
	fmt.Println(strings.Repeat("=", len(word)))
	fmt.Print(optLine)

	for {
		char := getChar()
		if strings.Index(allowedChars, char) == -1 {
			continue
		}
		return char
	}
}

// printMenu outputs list of pronunciations and handles user input. Return
// one of allowed keys
func printMenu(cfg Config, list []Pron, pronIdx int, isFirstWord, isLastWord bool) string {
	// format list of pronunciations
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

	// format user help status bar
	optLine := ""
	allowedChars := "erq"
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
		optLine += "[p]:previous word    "
		allowedChars += "p"
	}
	optLine += "[e]:enter a new word    [q]:quit\n"

	alreadySaid := false
UPDATE_PRINT:
	clearScreen()
	fmt.Print(pronLines)
	fmt.Print(optLine)

	// play pronunciation audio file
	if !alreadySaid {
		aPath := saveWord(cfg, list[pronIdx])
		sayWord(aPath)
		alreadySaid = true
	}

	// handle user input
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
func saveWord(cfg Config, item Pron) string {
	if cfg["VERBOSE"] == "yes" {
		fmt.Printf("Saving audio file: `%s`\n", item.aFile)
	}

	if cfg["CACHE"] == "yes" {
		_, err := os.Stat(item.cacheFile)
		if errors.Is(err, os.ErrNotExist) {
			err = getAudio(cfg, item.aURL, item.cacheFile)
			if err != nil {
				return ""
			}
		}

		if cfg["DOWNLOAD"] == "yes" {
			copyFile(cfg, item.cacheFile, item.aFile)
			return item.aFile
		}
		return item.cacheFile
	}

	// we do not use cache
	if cfg["DOWNLOAD"] == "yes" {
		err := getAudio(cfg, item.aURL, item.aFile)
		if err != nil {
			return ""
		}
	}

	// We have no cache and do not save file in local directory.
	// So we use temporary file if we are in interactive mode.
	// Otherwise we just do not need download anything
	if cfg["INTERACTIVE"] == "yes" {
		file := filepath.Join(tmpDir + filepath.Base(item.cacheFile))
		err := getAudio(cfg, item.aURL, file)
		if err != nil {
			return ""
		}
		return file
	}

	return ""
}

// pronCheck makes a seach request to be sure pronunciation for this word
// exists. I does not matter in case just one word, but if we have list of a few
// hundreds I am afraid we can be block by some anti-bot system
func pronCheck(cfg Config, word string) bool {
	if cfg["VERBOSE"] == "yes" {
		fmt.Printf("Checking pronunciation existing: `%s`\n", word)
	}

	pageURL := fmt.Sprintf("%s/search/%s/%s/", forvoURL, word, cfg["LANG"])
	pageText, err := getHTML(cfg, pageURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can not get search page for '%s'!\n", word)
		return false
	}

	// extract block with count of founded words
	countStr := `(?is)<section class="main_section">\s*<header>.*?` +
		`<p class="more">(.*?)</p>`
	countRe := regexp.MustCompile(countStr)
	count := countRe.FindStringSubmatch(pageText)
	if count == nil || count[1] == "0 words found" {
		return false
	}

	return true
}

// getPronList gets a pronunciation list for a specific word
func getPronList(cfg Config, word string) (result []Pron) {
	if cfg["VERBOSE"] == "yes" {
		fmt.Printf("Extracting pronunciation list for `%s`\n", word)
	}

	if cfg["PRONUNCIATION_CHECK"] == "yes" {
		if !pronCheck(cfg, word) {
			fmt.Fprintf(os.Stderr, "no pronunciations for '%s'!\n", word)
			return
		}
	}

	pageURL := fmt.Sprintf("%s/word/%s/#%s", forvoURL, word, cfg["LANG"])
	pageText, err := getHTML(cfg, pageURL)
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
		return
	}

	// extract every pronunciation <li> chunks
	pronStr := `(?is)<li.*?>(.*?)</li>`
	pronRe := regexp.MustCompile(pronStr)
	pronBlocks := pronRe.FindAllString(wordsBlock, -1)
	if pronBlocks == nil {
		fmt.Fprintln(os.Stderr, "can not extract separate pronunciations blocks")
		return
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

	chunkStr := `(?is)onclick="Play\(\d+,.*?,.*?,.*?,'(.*?)'.*?>\s*` +
		`Pronunciation by\s*(.*?)\s*` +
		`</span>\s*<span class="from">\((.*?)(?:\ from\ (.*?))?\)</span>`
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
	mp3String := string(decodedMp3)
	newLine := strings.LastIndex(mp3String, ".")
	if newLine > -1 {
		mp3String = mp3String[:newLine]
	}
	item.mp3 = audioURL + "/mp3/" + mp3String + ".mp3"
	item.ogg = audioURL + "/ogg/" + mp3String + ".ogg"

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
	if len(item.country) == 0 {
		item.country = "Unknown"
	}

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
