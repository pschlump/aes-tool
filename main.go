package main

// DONE - allow use of named pipe.
// 	https://github.com/nknorg/encrypted-stream
// 	https://tutorialedge.net/golang/go-encrypt-decrypt-aes-tutorial/
//
// mac/linux
//   $ mkfifo Nmae

// TODO File Retention
// TODO Copy to S3 for backup
// 		Script run to do retention
// 		Script run to do copy

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
var RotateHours = flag.Int("rotate-hours", 24, "how often to roate log files")
var RotateTemplate = flag.String("rotate-template", "./t1/logfile.%{timestamp%}.log.enc", "Template for output file name")
var BackupScript = flag.String("backup-log-files", "./bin/backup-log-files.sh", "Script to run to backup log files")

// file-pattern/name-template for generating new log files during rotation.
// 		Date Time - in order

// script to run to backup/cleanup files after rotation.
//		./bin/backup-log-files.sh "Path-To-Log-Files", "pattern-to-match"

var ch chan string = make(chan string, 1)
var timeout chan string = make(chan string, 2)
var wg sync.WaitGroup
var tokenTimeLeft int = 1
var n_tick int = 0
var oneHourInSeconds = 3600
var hourlyTimeLeft int = oneHourInSeconds
var httpServerList []*http.Server
var outputFilePtr *os.File = os.Stdout
var tokenCountdownMux *sync.Mutex = &sync.Mutex{}
var outputMux *sync.Mutex = &sync.Mutex{}

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

	nHoursLeft = *RotateHours

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

	if DbOn["log-rotate-test"] {
		oneHourInSeconds = 10
	}

	// -------------------------------------------------------------------------------------------------
	// Setup output file for non-pipe work.
	// -------------------------------------------------------------------------------------------------
	// if *output != "" && !*pipeInput {
	if *output != "" {
		outputMux.Lock()
		outputFilePtr, err = filelib.Fopen(*output, "w")
		if err != nil {
			outputMux.Unlock()
			fmt.Fprintf(os.Stderr, "Unable to open %s for output: %s\n", *output, err)
			os.Exit(1)
		}
		outputMux.Unlock()
		defer outputFilePtr.Close()
	}

	// TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO
	if DbOn["rotate-logs"] {
		RotateLogs()
		os.Exit(1)
	}

	// -------------------------------------------------------------------------------------------------
	// control interface (really only practical with ReadPipeForever)
	// -------------------------------------------------------------------------------------------------
	if *controlInterface != "" {
		dbgo.Fprintf(os.Stderr, "%(green)Control Interface Enabled. Listing at http://%s\n", *controlInterface)

		// ------------------------------------------------------------------------------
		// Main Processing
		// ------------------------------------------------------------------------------
		go OneSecondDispatch()

		// ticker on channel - send once a second
		Start1SecTimer()

		// ------------------------------------------------------------------------------
		// Setup signal capture
		// ------------------------------------------------------------------------------
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt)

		go func() {
			if DbOn["server-only"] {
				wg.Add(1)
			}
			mux := http.NewServeMux()

			mux.HandleFunc("/api/status", RespHandlerStatus)
			mux.HandleFunc("/api/v1/status", RespHandlerStatus)
			mux.HandleFunc("/api/v1/exit-server", RespHandleExitServer)
			mux.HandleFunc("/api/v1/rotate-logs", RespHandlerRotateLogs)
			mux.Handle("/", http.FileServer(http.Dir(*Dir)))

			httpServer := &http.Server{
				Addr:              *controlInterface,
				Handler:           mux,
				ReadTimeout:       5 * time.Second,
				WriteTimeout:      5 * time.Second,
				IdleTimeout:       90 * time.Second,
				ReadHeaderTimeout: 10 * time.Second,
			}
			httpServerList = append(httpServerList, httpServer)

			log.Fatal(httpServer.ListenAndServe())
		}()

		go func() {
			// ------------------------------------------------------------------------------
			// Catch signals from [Control-C]
			// ------------------------------------------------------------------------------
			select {
			case <-stop:
				fmt.Fprintf(os.Stderr, "\nShutting down the server... Received OS Signal...\n")
				// fmt.Fprintf(logFilePtr, "\nShutting down the server... Received OS Signal...\n")
				for _, httpServer := range httpServerList {
					ctx, cancel := context.WithTimeout(context.Background(), shutdownWaitTime*time.Second)
					defer cancel()
					err := httpServer.Shutdown(ctx)
					if err != nil {
						fmt.Printf("Error on shutdown: [%s]\n", err)
					}
				}
				os.Exit(0)
			}
		}()

	}

	if DbOn["server-only"] {
		dbgo.Printf("Waiting for serer only %(LF)\n")
		wg.Wait()
	}

	// -------------------------------------------------------------------------------------------------
	// Do encryption/decrypion work
	// -------------------------------------------------------------------------------------------------
	if *pipeInput && *encode != "" {

		dbgo.Printf("Forever read loop on pipe %(LF)\n")
		//		        In(pipe) Out(encrypted)
		ReadPipeForever(*encode, keyString)

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
		outputMux.Lock()
		outputFilePtr.WriteString(encContent)
		outputMux.Unlock()

	} else if *decode != "" {

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

			content, err := /*enc.*/ DataDecrypt(scanner.Text(), keyString)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to decrypt %s at line %d Error: %s\n", *output, line_no, err)
			} else {
				outputMux.Lock()
				fmt.Fprintf(outputFilePtr, "%s", content)
				outputMux.Unlock()
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
		for _, httpServer := range httpServerList {
			err := httpServer.Shutdown(ctx)
			if err != nil {
				fmt.Printf("Error on shutdown: [%s]\n", err)
			}
		}
	}()
}

//			http.HandleFunc("/api/v1/rotate-logs", RespHandlerRotateLogs)
func RespHandlerRotateLogs(www http.ResponseWriter, req *http.Request) {
	www.Header().Set("Content-Type", "application/json; charset=utf-8")

	www.WriteHeader(http.StatusOK) // 200
	fmt.Fprintf(www, `{"status":"not-implemented-yet"}`)

	// TODO ---------------------------------------------------------------------------------------------------------------
	// This is the file to rotate.
	// var out *os.File = os.Stdout
}

// OneSecondDispatch waits for a "kick" or a timeout and calls QrGenerate forever.
func OneSecondDispatch() {
	for {
		select {
		case <-ch:
			tokenCountdownMux.Lock()
			tokenTimeLeft = -1
			hourlyTimeLeft = 2
			tokenCountdownMux.Unlock()

		case <-timeout:
			tokenCountdownMux.Lock()
			tokenTimeLeft--
			hourlyTimeLeft--
			tokenCountdownMux.Unlock()
			if hourlyTimeLeft < 0 {
				tokenCountdownMux.Lock()
				hourlyTimeLeft = oneHourInSeconds
				tokenCountdownMux.Unlock()
				go RunHourlyProcessing()
			}
		}
	}
}

var nHoursLeft int = 24

func RunHourlyProcessing() {
	tokenCountdownMux.Lock()
	nHoursLeft--
	tokenCountdownMux.Unlock()
	if nHoursLeft < 0 {
		tokenCountdownMux.Lock()
		nHoursLeft = *RotateHours
		tokenCountdownMux.Unlock()
		RotateLogs()
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

func RotateLogs() {
	fmt.Printf("Rotate Logs Now\n")

	// var RotateTemplate = flag.String("rotate-template", "./t1/logfile.%{timestamp%}.log.enc", "Template for output file name")
	var err error

	t := time.Now()

	// generate new template (time stuff)
	mdata := make(map[string]string)
	mdata["timestamp"] = t.Format(time.RFC3339)
	// IP address
	// Host name
	// Read config file for data
	newFn := filelib.Qt(*RotateTemplate, mdata)

	outputMux.Lock()

	outputFilePtr.Close()

	err = os.Rename(*output, newFn) // 	rename existing file to be template-generated name
	if err != nil {
		outputMux.Unlock()
		fmt.Fprintf(os.Stderr, "Unable to rename %s to %s: %s\n", *output, newFn, err)
		os.Exit(1)
	}

	outputFilePtr, err = filelib.Fopen(*output, "w")
	if err != nil {
		outputMux.Unlock()
		fmt.Fprintf(os.Stderr, "Unable to open %s for output: %s\n", *output, err)
		os.Exit(1)
	}

	outputMux.Unlock()

	if *BackupScript != "" {

		// var BackupScript = flag.String("backup-log-files", "./bin/backup-log-files.sh", "Script to run to backup log files")
		out, err := RunCmdImpl(*BackupScript, []string{*output, newFn})

		dbgo.Printf("%s %s\n", out, err)

	}

	fmt.Printf("Rotate Logs End\n")
}

// const db7 = false

/* vim: set noai ts=4 sw=4: */
