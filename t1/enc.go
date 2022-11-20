package main

// FROM: https://tutorialedge.net/golang/go-encrypt-decrypt-aes-tutorial/

import (
	"fmt"
)

// var hex bool = false

func main() {

	text := []byte("My Super Secret Code Stuff")
	password := "Progress Is Slow Fast"

	// string := f ( []byte, string )
	s := EncBlocks(text, password)

	fmt.Printf("%s\n", s)
}
