package main

// FROM: https://tutorialedge.net/golang/go-encrypt-decrypt-aes-tutorial/

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

var hexEnabled bool = false

func main() {

	password := "Progress Is Slow Fast"
	key := HashStringsToByte(password)

	ciphertext_raw, err := ioutil.ReadFile(",a")
	if err != nil {
		fmt.Printf("Unable to read input file %s: error %s\n", ",a", err)
		os.Exit(1)
	}

	line := strings.TrimSuffix(string(ciphertext_raw), "\n") // remo
	// fmt.Printf("raw  ->%s<-\n", ciphertext_raw)
	// fmt.Printf("line ->%s<-\n", line)

	var ciphertext []byte
	if hexEnabled {
		ciphertext, err = hex.DecodeString(line)
		if err != nil {
			fmt.Println(err)
		}
	} else {
		ciphertext, err = base64.StdEncoding.DecodeString(line)
		if err != nil {
			fmt.Println(err)
		}
	}

	c, err := aes.NewCipher(key)
	if err != nil {
		fmt.Println(err)
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		fmt.Println(err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		fmt.Println(err)
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(plaintext))
}

func HashStringsToByte(a ...string) []byte {
	h := sha256.New()
	for _, z := range a {
		h.Write([]byte(z))
	}
	return []byte((h.Sum(nil)))
}
