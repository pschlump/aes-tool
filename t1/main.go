package main

// Quick demo of forever loop to read named pipe that can be closed
// and have EOF multiple times.

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
)

const BlockSize int = 1024

//var (
//	data  *os.File
//	part  []byte
//	err   error
//	count int
//)

func ReadNamedPipeForever(fileName string) (byteCount int, buffer *bytes.Buffer) {

	data, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer data.Close()

	reader := bufio.NewReader(data)
	buffer = bytes.NewBuffer(make([]byte, 0))
	part := make([]byte, BlockSize)

	var count int
	ii := 0
	for {
		ii++
		fmt.Printf("Read chunk %d\n", ii)
		if count, err = reader.Read(part); err != nil {
			fmt.Printf("err for break: %s\n", err)
			break
		}
		buffer.Write(part[:count]) // collect chuks into buffer
	}
	if err != io.EOF {
		log.Fatal("Error Reading ", fileName, ": ", err)
	} else {
		err = nil
	}

	byteCount = buffer.Len() // see len of buffer at end.
	return
}

func main() {
	for {
		length, _ := ReadNamedPipeForever("data.txt")
		fmt.Printf("length of file =%d\n", length)
	}
}
