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

	"github.com/pschlump/dbgo"
)

//func EncBlocks(text []byte, password string) string {
//
//	key := HashStringsToByte(password)
//
//	// Generate a new aes cipher using our 32 byte long key - Key length
//	// is guranteed via the hash.
//	c, err := aes.NewCipher(key)
//	// if there are any errors, handle them
//	if err != nil {
//		fmt.Printf("Unable to setup the encryption cyper: error %s\n", err)
//		os.Exit(1)
//	}
//
//	// gcm or Galois/Counter Mode, is a mode of operation
//	// for symmetric key cryptographic block ciphers
//	// - https://en.wikipedia.org/wiki/Galois/Counter_Mode
//	gcm, err := cipher.NewGCM(c)
//	if err != nil {
//		fmt.Fprintf(os.Stderr, "Unable to create NewGCM: %s\n", err)
//		os.Exit(1)
//	}
//
//	// creates a new byte array the size of the nonce
//	// which must be passed to Seal
//	nonce := make([]byte, gcm.NonceSize())
//	// populate our nonce with a random sequence
//	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
//		fmt.Fprintf(os.Stderr, "Unable to read data: %s\n", err)
//		os.Exit(1)
//	}
//
//	// here we encrypt our text using the Seal function
//	// Seal encrypts and authenticates plaintext, authenticates the
//	// additional data and appends the result to dst, returning the updated
//	// slice. The nonce must be NonceSize() bytes long and unique for all
//	// time, for a given key.
//
//	//	if hex {
//	//		fmt.Printf("%x\n", gcm.Seal(nonce, nonce, text, nil))
//	//	} else {
//	//		fmt.Printf("%s\n", base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, text, nil)))
//	//	}
//
//	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, text, nil))
//}

func HashStringsToByte(a ...string) []byte {
	h := sha256.New()
	for _, z := range a {
		h.Write([]byte(z))
	}
	return []byte((h.Sum(nil)))
}

func DecBlocks(ciphertext []byte, password string) (s string, err error) {

	key := HashStringsToByte(password)

	line := strings.TrimSuffix(string(ciphertext), "\n") // remo

	ciphertext, err = base64.StdEncoding.DecodeString(line)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to base64 decode: %s\n", err)
		return "", err
	}

	c, err := aes.NewCipher(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create NewCiphter: %s\n", err)
		return "", err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create NewGCM: %s\n", err)
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		fmt.Fprintf(os.Stderr, "ciphertext shorter than nonce, invalid length: %s\n", err)
		return "", err
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	// fmt.Fprintf("nonce %x\n", nonce)
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid Unencrypt - GCM did not match: %s\n", err)
		return "", err
	}

	// !! note !! this may not work with unicode characters -- have not tested it.
	i := 0
	for ; i < len(plaintext); i++ {
		if plaintext[i] == 0 {
			break
		}
	}
	plaintext = plaintext[0:i]

	return string(plaintext), nil

}

const BlockSize int = 1024

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
			break
		}
		s, _ := DataEncrypt(part, password)
		outputMux.Lock()
		fmt.Fprintf(outputFilePtr, "%s\n", s)
		outputMux.Unlock()
	}
	if err != io.EOF {
		log.Fatal("Error Reading ", inputFileName, ": ", err)
	} else {
		err = nil
	}

}

func ReadPipeForever(inputFileName, password string) (err error) {
	for {
		ReadNamedPipeToEOF(inputFileName, password)
	}
}

func DataDecrypt(encryptedString string, keyString string) (decrypted []byte, err error) {

	if db8 {
		dbgo.Printf("at:%(LF) ->%s<-\n", encryptedString)
	}
	key := HashPassword(keyString)

	encryptedString = fix0Instring(encryptedString)
	enc, err := base64.StdEncoding.DecodeString(encryptedString)
	if err != nil {
		dbgo.Printf("at:%(LF) err=%s\n", err)
		return
	}

	// Create a new Cipher Block from the key
	block, err := aes.NewCipher(key)
	if err != nil {
		dbgo.Printf("at:%(LF) err=%s\n", err)
		return
	}

	// Create a new GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		dbgo.Printf("at:%(LF) err=%s\n", err)
		return
	}

	//Get the nonce size
	nonceSize := aesGCM.NonceSize()

	// Extract the nonce from the encrypted data
	nonce, ciphertext := enc[:nonceSize], enc[nonceSize:]
	// fmt.Fprintf(os.Stderr, "from cypher: nonce %x\n", nonce)

	// Decrypt the data
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		dbgo.Printf("at:%(LF) err=%s\n", err)
		return
	}

	if db8 {
		dbgo.Printf("at:%(LF)\n")
	}

	// !! note !! this may not work with unicode characters -- have not tested it.
	i := 0
	for ; i < len(plaintext); i++ {
		if plaintext[i] == 0 {
			break
		}
	}
	plaintext = plaintext[0:i]

	return plaintext, nil
}

func fix0Instring(ee string) (rv string) {
	rv = ee
	// fmt.Fprintf(os.Stderr, "%x\n", ee)
	return
}

func HashPassword(a ...string) []byte {
	h := sha256.New()
	for _, z := range a {
		h.Write([]byte(z))
	}
	return h.Sum(nil)
}

func DataEncrypt(plaintext []byte, keyString string) (encryptedString string, err error) {

	if db11 {
		dbgo.Printf("at:%(LF)\n")
	}
	key := HashPassword(keyString)

	// Create a new Cipher Block from the using key
	block, err := aes.NewCipher(key)
	if err != nil {
		dbgo.Printf("at:%(LF) err=%s\n", err)
		return
	}

	// Create a new GCM - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	// See : https://golang.org/pkg/crypto/cipher/#NewGCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		// panic(err.Error())
		dbgo.Printf("at:%(LF) err=%s\n", err)
		return
	}

	// Create a nonce. Nonce should be from GCM
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		dbgo.Printf("at:%(LF) err=%s\n", err)
		return
	}

	// Encrypt the data using aesGCM.Seal
	// Since we don't want to save the nonce somewhere else in this case, we add it as a prefix to the
	// encrypted data. The first nonce argument in Seal is the prefix.
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)

	// Convert to base 64 string
	so := base64.StdEncoding.EncodeToString(ciphertext)

	if db11 {
		dbgo.Printf("at:%(LF)\n")
	}
	return so, nil
	// return fmt.Sprintf("%x", ciphertext), nil // xyzzy - change to base 64
}

const db8 = false
const db11 = false

/* vim: set noai ts=4 sw=4: */
