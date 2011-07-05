
package main

import (
        "net"
        "fmt"
        "os"
)

func main() {
    fmt.Println("listening on 8080..")
    local, err := net.Listen("tcp", ":8080")
    if local == nil {
        fatal("cannot listen: %v", err)
    }
    for {
        conn, err := local.Accept()
        if conn == nil {
            fatal("accept failed: %v", err)
        }
        go handle(conn)
    }
}

func handle(local net.Conn) {
    fmt.Println("Connection opened..")
    buf := make([]byte, 1024)
    local.SetReadTimeout(0)

    local.Write([]byte("Thanks for connecting\x00"))

    for {
        bytes,err := local.Read(buf)
        if err != nil {
            log("read failed: %v", err)
            return
        }
        if bytes > 0 {
            fmt.Printf("Received: %s\n", string(buf))
        }
    }
}

func log(s string, a ... interface{}) {
    fmt.Printf("%s\n", fmt.Sprintf(s, a))
}
func fatal(s string, a ... interface{}) {
    fmt.Fprintf(os.Stderr, "fatal: %s\n", fmt.Sprintf(s, a))
    os.Exit(2)
}
