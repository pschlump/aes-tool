
# aes-tool - A CLI tool for encrypting and decrypting text in a file.



<--
var encode = flag.String("encode", "", "file to encode")
var decode = flag.String("decode", "", "file to encode")
var output = flag.String("output", "", "file to encode")
var password = flag.String("password", "", "file read password from")
var help = flag.Bool("help", false, "print out usage message")
-->


## Example

Decrypt and send output to stdout.

```
$ aes-tool --decode file.encrypted
```
Decrypt and send output to a file

```
$ aes-tool --decode file.encrypted --output output.txt
```






## Test

Run `make test` to run tests.
