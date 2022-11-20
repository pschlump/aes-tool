package main

// FROM: https://tutorialedge.net/golang/go-encrypt-decrypt-aes-tutorial/

import (
	"fmt"
	"io/ioutil"
	"os"
)

func main() {

	password := "Progress Is Slow Fast"

	ciphertext, err := ioutil.ReadFile(",a")
	if err != nil {
		fmt.Printf("Unable to read input file %s: error %s\n", ",a", err)
		os.Exit(1)
	}

	// string := f ( []byte, string )
	plain := DecBlocks(ciphertext, password)

	fmt.Printf("%s\n", plain)
}
