package main
import (
        "net"
        "fmt"
        "os"
)

func main() {
    fmt.Println("listening on 843..")
    local, err := net.Listen("tcp", ":843")
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
    const expected_message = "<policy-file-request/>"
    buf := make([]byte, 128)

    local.Read(buf)

    if string(buf[0:len(expected_message)]) != expected_message {
        fmt.Printf("received malformed request: %s\n", string(buf))
    }

    // This is just a test script, otherwise this would go in a separate file..
    policyFile := "<?xml version=\"1.0\"?>" +
        "<!DOCTYPE cross-domain-policy SYSTEM \"/xml/dtds/cross-domain-policy.dtd\">" +
        "<!-- Policy file for xmlsocket://socks.example.com -->" +
        "<cross-domain-policy>" +
        "<!-- This is a master-policy file -->" +
        "<site-control permitted-cross-domain-policies=\"master-only\"/>" +
        "<allow-access-from domain=\"*\" to-ports=\"8080\" />" +
        "</cross-domain-policy>\x00"

    local.Write([]byte(policyFile))
}

func fatal(s string, a ... interface{}) {
    fmt.Fprintf(os.Stderr, "fatal: %s\n", fmt.Sprintf(s, a))
    os.Exit(2)
}

