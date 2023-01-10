package main

import (
	"fmt"
)

var confDir, confFile, cacheDir string

func main() {
	fmt.Println("Hello, world!\n")

	configInit()
}
