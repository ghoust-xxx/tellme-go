package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const confDirName = "tellme"
const confFileName = "config"
const cacheDirName = "cache"

type configFileValue struct {
	comment string
	key     string
	value   string
}

const configFileComment = "TellMe configuration file"

var configFileDefaults []configFileValue

// configInit makes sure config file and cache dir exist and read config values.
func configInit() {
	checkConfig()
	checkCache()

	fmt.Println(confDir)
	fmt.Println(confFile)
	fmt.Println(cacheDir)
}

// checkConfig checks if $HOME/.tellme/ || $XDG_CONFIG_HOME/tellme/,
// $HOME/.tellme/config || $XDG_CONFIG_HOME/tellme/config exist.
// If they are not tries to create them in $XDG paths.
func checkConfig() {
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

// setDefaultConfigValues define default value for creating a new config file
func setDefaultConfigValues() {
	configFileDefaults = []configFileValue{
		{
			comment: "interactive mode (yes|no)",
			key:     "INTERACTIVE",
			value:   "no",
		}, {
			comment: "check existence of pronunciation (yes|no)",
			key:     "PRONUNCIATION_CHECK",
			value:   "yes",
		}, {
			comment: "download audiofiles in current directory (yes|no)",
			key:     "DOWNLOAD",
			value:   "yes",
		}, {
			comment: "cache files (yes|no)",
			key:     "CACHE",
			value:   "yes",
		}, {
			comment: "cache directory. Default is $XDG_CACHE_HOME/tellme",
			key:     "CACHE_DIR",
			value:   "$XDG_CACHE_HOME/tellme",
		}, {
			comment: "language (en, es, de, etc)",
			key:     "LANG",
			value:   "nl",
		}, {
			comment: "audiofiles type (mp3|ogg)",
			key:     "TYPE",
			value:   "mp3",
		}, {
			comment: "verbose mode (yes|no)",
			key:     "VERBOSE",
			value:   "no",
		},
	}
}

// createNewConf write a new config file with all default values and comments
func createNewConf() {
	fmt.Println("create a new conf file")

	f, err := os.OpenFile(confFile, os.O_RDWR|os.O_CREATE, 0750)
	if err != nil {
		log.Fatal(err)
	}

	setDefaultConfigValues()

	defer func() {
		err = f.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	var defaultConfig strings.Builder

	defaultConfig.WriteString(fmt.Sprintf("# %s\n\n", configFileComment))
	for i := 0; i < len(configFileDefaults); i++ {
		defaultConfig.WriteString(fmt.Sprintf(
			"# %s\n%s=%s\n\n",
			configFileDefaults[i].comment,
			configFileDefaults[i].key,
			configFileDefaults[i].value,
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
