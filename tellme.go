package main

import (
	"fmt"
)

var confDir, confFile, cacheDir string

const confDirName = "tellme"
const confFileName = "config"
const cacheDirName = "cache"

func main() {
	fmt.Println("Hello, world!\n")

	configInit()
	fmt.Println(confDir)
	fmt.Println(confFile)
	fmt.Println(cacheDir)
}

