package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const confDirName = "tellme"
const confFileName = "config"
const cacheDirName = "cache"

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

// createNewConf write a new config file with all default values and comments
func createNewConf() {
	fmt.Println("create a new conf file")

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

	defaultConfig :=
		`# TellMe configuration file

# interactive mode (yes|no)
INTERACTIVE=no

# check existence of pronunciation (yes|no)
PRONUNCIATION_CHECK=yes

# download audiofiles in current directory (yes|no)
DOWNLOAD=yes

# cache files (yes|no)
CACHE=yes

# cache directory. Default is \$HOME/.tellme/cache
CACHE_DIR=\$HOME/.tellme/cache

# language (en, es, de, etc). Default is 'nl'
LANG=nl

# audiofiles type (mp3|ogg). mp3 by default
TYPE=mp3

# verbose mode (yes|no)
VERBOSE=no

`
	_, err = f.WriteString(defaultConfig)
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
