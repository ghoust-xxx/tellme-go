package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type configFileValue struct {
	comment string
	key     string
	value   string
	fname   string
	ftype   string
}

const confDirName = "tellme"
const confFileName = "config"
const cacheDirName = "cache"
const configFileComment = "TellMe configuration file"

var configDefaults []configFileValue
var confDir, confFile, cacheDir string

// configInit makes sure config file and cache dir exist and read config values.
func configInit() {
	checkConfig()
	setConfigValues()
	checkCache()
}

// setConfigValues set all config value that app will be use.
// It set them it this priority: default values, values from config file and
// at last values from command line paramas.
func setConfigValues() {
	setDefaultConfigValues()
	updateFromConfigFile()
	updateFromCmdLine()
	optionsValidation()
}

// optionsValidation check if current flag combination is allowed
func optionsValidation() {
	if len(os.Args) > 0 && cfg["FILE"] != "" {
		log.Fatal("you can use only --file options or words in command line, not both")
	}
}

// updateFromCmdLine get command line params and update app config values
// accordingly.
func updateFromCmdLine() {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	for _, val := range configDefaults {
		switch val.ftype {
		case "yesno":
			fs.Func(val.fname, val.comment, buildYesNo(val))
		case "path":
			fs.Func(val.fname, val.comment, buildPath(val))
		case "lang":
			fs.Func(val.fname, val.comment, buildLang(val))
		case "aformat":
			fs.Func(val.fname, val.comment, buildAFormat(val))
		default:
			panic("Wrong config type (" + val.ftype + "). This should never happen")
		}
	}
	pFile := fs.String("f", "", "read input from `filename`")
	fs.Parse(os.Args[1:])
	cfg["FILE"] = *pFile
	os.Args = fs.Args()
}

// buildYesNo parses [yes | no] args type
func buildYesNo(val configFileValue) func(s string) error {
	return func(s string) error {
		if s == "yes" || s == "no" {
			cfg[val.key] = s
			return nil
		}
		return errors.New("have to be yes or no")
	}
}

// buildPath parses path args type
func buildPath(val configFileValue) func(s string) error {
	return func(s string) error {
		cfg[val.key] = s
		return nil
	}
}

// buildLang parses lang args type
func buildLang(val configFileValue) func(s string) error {
	return func(s string) error {
		if len(s) != 2 {
			return errors.New("have to be 2 letters language code")
		}
		cfg[val.key] = s
		return nil
	}
}

// buildAFormat parses audio format args type
func buildAFormat(val configFileValue) func(s string) error {
	return func(s string) error {
		if s == "mp3" || s == "ogg" {
			cfg[val.key] = s
			return nil
		}
		return errors.New("have to be mp3 or ogg")
	}
}

// updateFromConfigFile read config file and updates app config values
// accordingly.
func updateFromConfigFile() {
	cFile, err := os.Open(confFile)
	if err != nil {
		log.Fatal(err)
	}
	defer cFile.Close()

	iniLine := regexp.MustCompile(`^\s*(\w+)=([\w\./\\]+)\s*`)
	var cnt int
	scanner := bufio.NewScanner(cFile)
	for scanner.Scan() {
		cnt++
		line := []rune(scanner.Text())
		if len(line) == 0 {
			continue
		}
		if line[0] == '#' {
			continue
		}

		matches := iniLine.FindStringSubmatch(string(line))
		if matches == nil {
			log.Fatal(
				fmt.Sprintf("error in config file %v, line %v", confFile, cnt))
		}

		key := matches[1]
		value := matches[2]
		if _, ok := cfg[key]; !ok {
			log.Fatal(
				fmt.Sprintf("nonexisting key in config file %v, line %v",
					confFile, cnt))
		}

		cfg[key] = value
	}
	if err = scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

// setDefaultConfigValues set config value in case if some missing both in the
// config file and in command line params.
func setDefaultConfigValues() {
	cfg = make(Config)
	for _, val := range configDefaults {
		cfg[val.key] = val.value
	}
}

// checkConfig checks if $HOME/.tellme/ || $XDG_CONFIG_HOME/tellme/,
// $HOME/.tellme/config || $XDG_CONFIG_HOME/tellme/config exist.
// If they are not tries to create them in $XDG paths.
func checkConfig() {
	setDefaultConfigFileValues()

	// Check if configuration directory already exists.
	// Create it if it does not.
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	userConfDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}

	confDir = filepath.Join(userHomeDir, "."+confDirName)
	_, err = os.Stat(confDir)
	if errors.Is(err, os.ErrNotExist) {
		confDir = filepath.Join(userConfDir, confDirName)
		_, err = os.Stat(confDir)
		if errors.Is(err, os.ErrNotExist) {
			err = os.Mkdir(confDir, 0750)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	// Check if configuration file already exists.
	// Create it if it does not.
	confFile = filepath.Join(confDir, confFileName)
	_, err = os.Stat(confFile)
	if errors.Is(err, os.ErrNotExist) {
		createNewConf()
	}
}

// setDefaultConfigFileValues define default value for creating a new config file
func setDefaultConfigFileValues() {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Fatal(err)
	}

	configDefaults = []configFileValue{
		{
			comment: "interactive mode `[yes | no]`. Default no",
			key:     "INTERACTIVE",
			value:   "no",
			fname:   "i",
			ftype:   "yesno",
		}, {
			comment: "check existence of pronunciation `[yes | no]`. Default yes",
			key:     "PRONUNCIATION_CHECK",
			value:   "yes",
			fname:   "check",
			ftype:   "yesno",
		}, {
			comment: "download audiofiles in current directory `[yes | no]`. Default yes",
			key:     "DOWNLOAD",
			value:   "yes",
			fname:   "d",
			ftype:   "yesno",
		}, {
			comment: "cache files `[yes | no]`. Default yes",
			key:     "CACHE",
			value:   "yes",
			fname:   "c",
			ftype:   "yesno",
		}, {
			comment: "cache directory `[any valid path]`. Default " + userCacheDir + "/tellme",
			key:     "CACHE_DIR",
			value:   userCacheDir + "/tellme",
			fname:   "cache-dir",
			ftype:   "path",
		}, {
			comment: "language `[en | es | de | etc]`. Default nl",
			key:     "LANG",
			value:   "nl",
			fname:   "l",
			ftype:   "lang",
		}, {
			comment: "audiofiles type `[mp3 | ogg ]`. Default mp3",
			key:     "ATYPE",
			value:   "mp3",
			fname:   "t",
			ftype:   "aformat",
		}, {
			comment: "verbose mode [yes | no]. Default no",
			key:     "VERBOSE",
			value:   "no",
			fname:   "verbose",
			ftype:   "yesno",
		},
	}
}

// createNewConf write a new config file with all default values and comments
func createNewConf() {
	f, err := os.OpenFile(confFile, os.O_RDWR|os.O_CREATE, 0640)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		err = f.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	var defaultConfig strings.Builder

	defaultConfig.WriteString(fmt.Sprintf("# %s\n\n", configFileComment))
	for i := 0; i < len(configDefaults); i++ {
		defaultConfig.WriteString(fmt.Sprintf(
			"# %s\n%s=%s\n\n",
			configDefaults[i].comment,
			configDefaults[i].key,
			configDefaults[i].value,
		))
	}

	_, err = f.WriteString(defaultConfig.String())
	if err != nil {
		log.Fatal(err)
	}
}

// checkCache checks if $HOME/.tellme/cache/ || $XDG_CACHE_HOME/tellme/ exists.
// If it is not tries to create them in $XDG paths.
func checkCache() {
	// Check if cache directory already exists.
	// Create it if it does not.
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Fatal(err)
	}

	cacheDir = filepath.Join(userHomeDir, "."+confDirName, cacheDirName)
	_, err = os.Stat(cacheDir)
	if errors.Is(err, os.ErrNotExist) {
		cacheDir = filepath.Join(userCacheDir, confDirName)
		_, err = os.Stat(cacheDir)
		if errors.Is(err, os.ErrNotExist) {
			err = os.Mkdir(cacheDir, 0750)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
