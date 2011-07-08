package main

import (
	"net"
	"fmt"
	"os"
)

func main() {
	fmt.Println("connecting to 8080..")
	local, err := net.Dial("tcp", "localhost:8080")
	if local == nil {
		fatal("cannot connect: %v", err)
	}
	local.Write([]byte("Hello there"))
}

func fatal(s string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "fatal: %s\n", fmt.Sprintf(s, a))
	os.Exit(2)
}
