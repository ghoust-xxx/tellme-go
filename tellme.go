package main

import "os"

func main() {
	cfg := configInit()
	mainLoop(cfg, os.Args)
}
