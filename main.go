package main

// TODO - allow use of named pipe.
// https://github.com/nknorg/encrypted-stream
// https://tutorialedge.net/golang/go-encrypt-decrypt-aes-tutorial/
//
// mac
//   $ mkfifo Nmae

// TODO - implement log rotation

import (
	"bufio"
	"flag"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"syscall"

	"github.com/pschlump/filelib"
	"github.com/pschlump/qr-secret/enc"
	"golang.org/x/term"
)

var pipeInput = flag.Bool("pipe-input", false, "Input is a named pipe")
var encode = flag.String("encode", "", "file to encode")
var decode = flag.String("decode", "", "file to decode")
var output = flag.String("output", "", "file to send output to")
var password = flag.String("password", "", "file read password from")
var help = flag.Bool("help", false, "print out usage message")

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "aes-tool: Usage: %s [flags]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse() // Parse CLI arguments to this, --cfg <name>.json

	fns := flag.Args()
	if len(fns) != 0 {
		fmt.Printf("Extra arguments are not supported [%s]\n", fns)
		os.Exit(1)
	}

	if *help {
		flag.Usage()
		os.Exit(1)
	}
	if *encode == "" && *decode == "" {
		fmt.Printf("Failed to specify --encode or --decode flag.  Must have one of them\n")
		flag.Usage()
		os.Exit(1)
	}

	var keyString string
	var err error
	var out *os.File

	if *password == "" || *password == "-" {
		keyString, err = ReadPassword(*output != "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s on reading password\n", err)
			os.Exit(1)
		}
	} else if len(*password) > 10 && strings.HasPrefix(*password, "!env!") {
		p := os.Getenv((*password)[len("!env!"):])
		// fmt.Printf("got ->%s<-\n", p)
		password = &p
	} else {
		buf, err := ioutil.ReadFile(*password)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: unable to read password input file %s, error:%s\n", *password, err)
			os.Exit(1)
		}
		keyString = strings.Trim(string(buf), "\n\r \t")
	}

	out = os.Stdout
	if *output != "" && !*pipeInput {
		out, err = filelib.Fopen(*output, "w")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to open %s for output: %s\n", *output, err)
			os.Exit(1)
		}
		defer out.Close()
	}

	if *pipeInput && *encode != "" {
		// TODO -----------------------------------------------------------------------------------------------------------------------

		//				   In(pipe) Out(encrypted)
		ReadPipeForever(*encode, *output, *password)

	} else if *encode != "" {

		buf, err := ioutil.ReadFile(*encode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s on %s\n", err, *encode)
			os.Exit(1)
		}

		content := string(buf)

		encContent, err := enc.DataEncrypt([]byte(content), keyString)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to encrypt %s Error: %s\n", *output, err)
			os.Exit(1)
		}

		// Output encrypted data
		out.WriteString(encContent)

	} else if *decode != "" {

		encContent, err := ioutil.ReadFile(*decode)
		if err != nil {
			os.Exit(1)
		}
		content, err := enc.DataDecrypt(string(encContent), keyString)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to decrypt %s Error: %s\n", *output, err)
			os.Exit(1)
		}
		fmt.Fprintf(out, "%s", content)

	}
}

func ReadPassword(prompt bool) (password string, err error) {

	if prompt {
		fmt.Print("Enter Password: ")
	}
	if runtime.GOOS == "windows" {

		reader := bufio.NewReader(os.Stdin)
		password, err = reader.ReadString('\n')
		if err != nil {
			return "", err
		}

	} else {

		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return "", err
		}

		password = string(bytePassword)

	}

	return strings.TrimSpace(password), nil
}

const db7 = false

/* vim: set noai ts=4 sw=4: */
