package main

// FROM: https://tutorialedge.net/golang/go-encrypt-decrypt-aes-tutorial/

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

var hex bool = false

func main() {

	text := []byte("My Super Secret Code Stuff")
	password := "Progress Is Slow Fast"
	key := HashStringsToByte(password)

	// Generate a new aes cipher using our 32 byte long key - Key length
	// is guranteed via the hash.
	c, err := aes.NewCipher(key)
	// if there are any errors, handle them
	if err != nil {
		fmt.Printf("Unable to setup the encryption cyper: error %s\n", err)
		os.Exit(1)
	}

	// gcm or Galois/Counter Mode, is a mode of operation
	// for symmetric key cryptographic block ciphers
	// - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		fmt.Println(err)
	}

	// creates a new byte array the size of the nonce
	// which must be passed to Seal
	nonce := make([]byte, gcm.NonceSize())
	// populate our nonce with a random sequence
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		fmt.Println(err)
	}

	// here we encrypt our text using the Seal function
	// Seal encrypts and authenticates plaintext, authenticates the
	// additional data and appends the result to dst, returning the updated
	// slice. The nonce must be NonceSize() bytes long and unique for all
	// time, for a given key.

	if hex {
		fmt.Printf("%x\n", gcm.Seal(nonce, nonce, text, nil))
	} else {
		fmt.Printf("%s\n", base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, text, nil)))
	}
}

func HashStringsToByte(a ...string) []byte {
	h := sha256.New()
	for _, z := range a {
		h.Write([]byte(z))
	}
	return []byte((h.Sum(nil)))
}
