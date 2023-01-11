package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

type configFileValue struct {
	comment string
	key     string
	value   string
}

const confDirName = "tellme"
const confFileName = "config"
const cacheDirName = "cache"
const configFileComment = "TellMe configuration file"

var configFileDefaults []configFileValue
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
	fmt.Println(cfg)
}

// updateFromCmdLine get command line params and update app config values
// accordingly.
func updateFromCmdLine() {
}

// updateFromConfigFile read config file and updates app config values
// accordingly.
func updateFromConfigFile() {
	cFile, err := ini.Load(confFile)
	if err != nil {
		log.Fatal(err)
	}

	for _, key := range cFile.Section("").Keys() {
		cfg[key.Name()] = key.Value()
	}
}

// setDefaultConfigValues set config value in case if some missing both in the
// config file and in command line params.
func setDefaultConfigValues() {
	cfg = make(map[string]string)
	for _, val := range configFileDefaults {
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
			value:   userCacheDir + "/tellme",
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
	f, err := os.OpenFile(confFile, os.O_RDWR|os.O_CREATE, 0750)
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
