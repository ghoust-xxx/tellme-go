package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var confDir, confFile, cacheDir string

const confDirName = "tellme"
const confFileName = "config"
const cacheDirName = "cache"

func main() {
	fmt.Println("Hello, world!\n")

	config_init()
}

// config_init checks if $HOME/.tellme/ || $XDG_CONFIG_HOME/tellme/,
// $HOME/.tellme/config || $XDG_CONFIG_HOME/tellme/config,
// $HOME/.tellme/cache/ || $XDG_CACHE_HOME/tellme/ exists
// If they are not tries to create them in $XDG paths.
func config_init() {
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

	// Check if cache directory already exists.
	// Create it if it does not.
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

	// Check if configuration file already exists.
	// Create it if it does not.
	confFile = filepath.Join(confDir, confFileName)
	_, err = os.Stat(confFile)
	if errors.Is(err, os.ErrNotExist) {
		// TODO: create a new conf file
		fmt.Println("create a new conf file")
	}

	fmt.Println(confDir)
	fmt.Println(confFile)
	fmt.Println(cacheDir)
}
