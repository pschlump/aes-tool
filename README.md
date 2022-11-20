
# aes-tool - A command line (CLI) tool for encrypting and decrypting text in a file.


This tool is specifically designed to work with log files and other snippets
that have been encrypted. 

An example of an AES encrypted chunk of data is an encrypted QR code.


<--
var encode = flag.String("encode", "", "file to encode")
var decode = flag.String("decode", "", "file to encode")
var output = flag.String("output", "", "file to encode")
var password = flag.String("password", "", "file read password from")
var help = flag.Bool("help", false, "print out usage message")
-->


## Examples

Decrypt and send output to stdout.  Take the password from the environment
variable `ENC_AES_TOOL`.

```
$ aes-tool --decode file.encrypted --password "#env#ENC_AES_TOOL"
```

Decrypt and send output to a file.   Prompt for the password.

```
$ aes-tool --decode file.encrypted --output output.txt
```

Read a named pipe for data and encrypt it.  In this form it will reaa in blocks
forever and send encrypted data to the output.

```
$ aes-tool --encode ./t1/data.txt --pipe-input --output ./out/encrypted-log.enc --password "#env#SENDGRID_API_KEY" &
```




## Test

Run `make test` to run tests.

