package main

// DONE - allow use of named pipe.
// 	https://github.com/nknorg/encrypted-stream
// 	https://tutorialedge.net/golang/go-encrypt-decrypt-aes-tutorial/
//
// mac/linux
//   $ mkfifo Nmae

// TODO - implement log rotation

// TODO - implement a HTTP control interface
//		--control-interface 127.0.0.1:14202
//		Starts system with an api of
//			/api/v1/status
//			/api/v1/exit-server
//			/api/v1/rotate-logs-now

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pschlump/dbgo"
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
var controlInterface = flag.String("control-interface", "", "turn on HTTP control interface")
var Dir = flag.String("dir", "./www", "directory to server static files from")

var ch chan string
var timeout chan string

// var timeout chan string
// var ch chan string
var tokenCountdownMux *sync.Mutex
var tokenTimeLeft int
var n_tick int = 0
var hourlyTimeLeft int = 3600

func init() {
	ch = make(chan string, 1)
	timeout = make(chan string, 2)
	tokenTimeLeft = 1
	timeout = make(chan string, 1)
	tokenCountdownMux = &sync.Mutex{}
}

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

	// -------------------------------------------------------------------------------------------------
	// save pidfile.
	// -------------------------------------------------------------------------------------------------
	pid := os.Getpid()
	os.MkdirAll("./aes-tool-log", 0755)
	ioutil.WriteFile("./aes-tool-log/pidfile.log", []byte(fmt.Sprintf("%d\n", pid)), 0644)

	// -------------------------------------------------------------------------------------------------
	// Get Password / Passphrase
	// -------------------------------------------------------------------------------------------------
	var keyString string
	var err error
	var out *os.File

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
	// control interface (really only practical with ReadPipeForever)
	// -------------------------------------------------------------------------------------------------
	if *controlInterface != "" {
		dbgo.Fprintf(os.Stderr, "%(green)Control Interface Enabled. Listing at http://%s\n", *controlInterface)
		go func() {

			// ------------------------------------------------------------------------------
			// Main Processing
			// ------------------------------------------------------------------------------
			// go TimedDispatch()
			go OneSecondDispatch()

			// ticker on channel - send once a minute
			Start1SecTimer()

			// ------------------------------------------------------------------------------
			// Setup signal capture
			// ------------------------------------------------------------------------------
			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt)

			http.HandleFunc("/api/status", RespHandlerStatus)
			http.HandleFunc("/api/v1/status", RespHandlerStatus)
			http.HandleFunc("/api/v1/exit-server", RespHandleExitServer)
			http.HandleFunc("/api/v1/rotate-logs", RespHandlerRotateLogs)
			http.Handle("/", http.FileServer(http.Dir(*Dir)))

			log.Fatal(http.ListenAndServe(*controlInterface, nil))
		}()
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

// RespHandlerStatus reports a JSON status.
func RespHandlerStatus(www http.ResponseWriter, req *http.Request) {
	q := req.RequestURI

	var rv string
	www.Header().Set("Content-Type", "application/json; charset=utf-8")
	rv = fmt.Sprintf(`{"status":"success","name":"aes-tool version 1.0.0","URI":%q,"req":%s, "response_header":%s}`, q, dbgo.SVarI(req), dbgo.SVarI(www.Header()))

	io.WriteString(www, rv)
}

const shutdownWaitTime = 1

// HandleExitServer - graceful server shutdown.
func RespHandleExitServer(www http.ResponseWriter, req *http.Request) {
	pid := os.Getpid()
	//if ymux.IsTLS(req) {
	//	www.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	//}
	www.Header().Set("Content-Type", "application/json; charset=utf-8")

	www.WriteHeader(http.StatusOK) // 200
	fmt.Fprintf(www, `{"status":"success", "pid":%v}`, pid)

	go func() {
		// Implement graceful exit with auth_key
		fmt.Fprintf(os.Stderr, "\nShutting down the server... Received /exit-server?auth_key=...\n")
		ctx, cancel := context.WithTimeout(context.Background(), shutdownWaitTime*time.Second)
		defer cancel()
		_ = ctx
		/*xyzzy
		err := httpServer.Shutdown(ctx)
		if err != nil {
			fmt.Printf("Error on shutdown: [%s]\n", err)
		}
		*/
	}()
}

//			http.HandleFunc("/api/v1/rotate-logs", RespHandlerRotateLogs)
func RespHandlerRotateLogs(www http.ResponseWriter, req *http.Request) {
	// TODO ---------------------------------------------------------------------------------------------------------------
	www.Header().Set("Content-Type", "application/json; charset=utf-8")

	www.WriteHeader(http.StatusOK) // 200
	fmt.Fprintf(www, `{"status":"not-implemented-yet"}`)
}

// OneSecondDispatch waits for a "kick" or a timeout and calls QrGenerate forever.
func OneSecondDispatch() {
	for {
		select {
		case <-ch:
			tokenCountdownMux.Lock()
			tokenTimeLeft = -1
			// hourlyTimeLeft = 2
			tokenCountdownMux.Unlock()
			// proxyApi.GuranteeCurrentToken()

		case <-timeout:
			tokenCountdownMux.Lock()
			tokenTimeLeft--
			// hourlyTimeLeft--
			tokenCountdownMux.Unlock()
			//if hourlyTimeLeft < 0 {
			//	tokenCountdownMux.Lock()
			//	hourlyTimeLeft = 3600
			//	tokenCountdownMux.Unlock()
			//	go RunHourlyProcessing()
			//}
		}
	}
}

func Start1SecTimer() {
	// ticker on channel - send once a second
	go func(n int) {
		for {
			time.Sleep(time.Duration(n) * time.Second)
			SendTimeout() // timeout <- "timeout"
		}
	}(1)
}

func SendTimeout() {
	timeout <- "timeout"
}

func SendKick() {
	ch <- "kick" // on control-channel - send "kick"
}

const db7 = false

/* vim: set noai ts=4 sw=4: */
