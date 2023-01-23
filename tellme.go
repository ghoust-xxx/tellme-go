package main

import "os"

type Config map[string]string

var cfg Config

func main() {
	configInit()
	mainLoop(cfg, os.Args)
}
