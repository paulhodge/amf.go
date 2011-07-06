
package main

import (
        "net"
        "fmt"
        "os"
)

import amf "../protocol"

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

type GrowableWriter struct {
    data []byte
}

func (self *GrowableWriter) Write(d []byte) (n int, err os.Error) {
    self.data = append(self.data, d...)
    return len(d), nil
}

func handle(local net.Conn) {
    fmt.Println("Connection opened..")

    // Socket.mxml will send a bunch of AMF3 encoded values, each preceded by
    // a string label.

    outgoing := &GrowableWriter{}

    for {
        label,err := amf.ReadString(local)
        if label == "" || err != nil {
            fmt.Println("Received empty label")
            break
        }
        obj, err := amf.ReadValueAmf3(local)
        if err != nil {
            fmt.Printf("%v\n", err)
            break
        }
        fmt.Printf("%s %v\n",label, obj)

        fmt.Printf("writing label: %s\n", label)

        // Write the value to our outgoing buffer.
        amf.WriteString(outgoing, label)
        amf.WriteValueAmf3(outgoing, obj)
    }

    // Write all of our data, prepended with size.
    amf.WriteInt32(local, int32(len(outgoing.data)))
    fmt.Printf("sending %d bytes\n", len(outgoing.data))

    local.Write(outgoing.data)
    amf.WriteString(local, "")
}

func log(s string, a ... interface{}) {
    fmt.Printf("%s\n", fmt.Sprintf(s, a))
}
func fatal(s string, a ... interface{}) {
    fmt.Fprintf(os.Stderr, "fatal: %s\n", fmt.Sprintf(s, a))
    os.Exit(2)
}
