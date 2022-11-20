package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/pschlump/filelib"
)

func EncBlocks(text []byte, password string) string {

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

	//	if hex {
	//		fmt.Printf("%x\n", gcm.Seal(nonce, nonce, text, nil))
	//	} else {
	//		fmt.Printf("%s\n", base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, text, nil)))
	//	}

	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, text, nil))
}

func HashStringsToByte(a ...string) []byte {
	h := sha256.New()
	for _, z := range a {
		h.Write([]byte(z))
	}
	return []byte((h.Sum(nil)))
}

func DecBlocks(ciphertext []byte, password string) string {

	key := HashStringsToByte(password)

	line := strings.TrimSuffix(string(ciphertext), "\n") // remo

	ciphertext, err := base64.StdEncoding.DecodeString(line)
	if err != nil {
		fmt.Println(err)
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

	return string(plaintext)

}

const BlockSize int = 1024

var outputFilePtr *os.File = os.Stdout

var DbOn map[string]bool = make(map[string]bool)

func ReadNamedPipeToEOF(inputFileName, password string) {

	data, err := os.Open(inputFileName)
	if err != nil {
		log.Fatal(err)
	}
	defer data.Close()

	reader := bufio.NewReader(data)
	part := make([]byte, BlockSize)

	ii := 0
	for {
		ii++
		if DbOn["db1"] {
			fmt.Printf("Read chunk %d\n", ii)
		}
		if _, err = reader.Read(part); err != nil {
			// fmt.Printf("err for break: %s\n", err)
			break
		}
		s := EncBlocks(part, password)
		fmt.Fprintf(outputFilePtr, "%s\n", s)
	}
	if err != io.EOF {
		log.Fatal("Error Reading ", inputFileName, ": ", err)
	} else {
		err = nil
	}

}

func ReadPipeForever(inputFileName, outputFileName, password string) (err error) {
	outputFilePtr, err = filelib.Fopen(outputFileName, "w")
	if err != nil {
		fmt.Printf("Unable to open %s for output: %s\n", outputFileName, err)
		return
	}
	for {
		ReadNamedPipeToEOF(inputFileName, password)
	}
}
