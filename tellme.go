package main

import (
	"fmt"

	"github.com/adrg/xdg"
)


func main() {
	fmt.Println("Hello, world!")
	fmt.Println(xdg.ConfigHome)
}
