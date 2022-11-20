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
	"log"
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
var debugFlag = flag.String("debug-flag", "", "enable debug flags")

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

	if *debugFlag != "" {
		for _, vv := range strings.Split(*debugFlag, ",") {
			DbOn[vv] = true
		}
	}

	if *encode == "" && *decode == "" {
		fmt.Printf("Failed to specify --encode or --decode flag.  Must have one of them\n")
		flag.Usage()
		os.Exit(1)
	}

	var keyString string
	var err error
	var out *os.File

	// -------------------------------------------------------------------------------------------------
	// Get Password / Passphrase
	// -------------------------------------------------------------------------------------------------
	if *password == "" || *password == "-" {
		keyString, err = ReadPassword(*output != "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s on reading password\n", err)
			os.Exit(1)
		}
	} else if len(*password) > 10 && strings.HasPrefix(*password, "#env#") {
		p := os.Getenv((*password)[len("#env#"):])
		if DbOn["echo-password-env"] {
			fmt.Fprintf(os.Stderr, "password = (from env) ->%s<-\n", p)
		}
		keyString = p
	} else {
		buf, err := ioutil.ReadFile(*password)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: unable to read password input file %s, error:%s\n", *password, err)
			os.Exit(1)
		}
		keyString = strings.Trim(string(buf), "\n\r \t")
	}
	// 	dbgo.Fprintf(os.Stderr, "%(red)->%s<-\n", keyString)

	if DbOn["dump-debug-flag"] {
		for i, v := range DbOn {
			fmt.Fprintf(os.Stderr, "DbOn[%v] = %v\n", i, v)
		}
	}

	// -------------------------------------------------------------------------------------------------
	// Setup output file for non-pipe work.
	// -------------------------------------------------------------------------------------------------
	out = os.Stdout
	if *output != "" && !*pipeInput {
		out, err = filelib.Fopen(*output, "w")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to open %s for output: %s\n", *output, err)
			os.Exit(1)
		}
		defer out.Close()
	}

	// -------------------------------------------------------------------------------------------------
	// Do encryption/decrypion work
	// -------------------------------------------------------------------------------------------------
	if *pipeInput && *encode != "" {

		//		        In(pipe) Out(encrypted)
		ReadPipeForever(*encode, *output, keyString)

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

		// OLD Code : the way this used to work
		//	encContent, err := ioutil.ReadFile(*decode)
		//	if err != nil {
		//		os.Exit(1)
		//	}
		//	content, err := enc.DataDecrypt(string(encContent), keyString)
		//	if err != nil {
		//		fmt.Fprintf(os.Stderr, "Unable to decrypt %s Error: %s\n", *output, err)
		//		os.Exit(1)
		//	}
		//	fmt.Fprintf(out, "%s", content)

		ifp, err := os.Open(*decode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to open %s for input: %s\n", *decode, err)
			os.Exit(1)
		}
		defer ifp.Close()

		scanner := bufio.NewScanner(ifp)
		line_no := 0
		for scanner.Scan() {
			line_no++
			if DbOn["echo-raw-input"] {
				fmt.Fprintf(os.Stderr, "%5d:->%s<-\n", line_no, scanner.Text())
			}

			//content, err := DecBlocks([]byte(scanner.Text()), keyString)
			//if err != nil {
			//	fmt.Fprintf(os.Stderr, "Falined on line %s with %s\n", line_no, err)
			//} else {
			//	fmt.Fprintf(out, "%s", content)
			//}
			content, err := /*enc.*/ DataDecrypt(scanner.Text(), keyString)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to decrypt %s at line %d Error: %s\n", *output, line_no, err)
			} else {
				fmt.Fprintf(out, "%s", content)
			}
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
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
