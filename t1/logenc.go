package main

// Quick demo of forever loop to read named pipe that can be closed
// and have EOF multiple times.

func main() {
	password := "Pink Butterflies"
	ReadPipeForever("data.txt", "data.txt.enc", password)
}
