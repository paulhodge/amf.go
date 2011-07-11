package main

import (
	"net"
	"fmt"
	"os"
)

//import amf "../protocol"

const listen_on = ":8080"
const redirect_to = "localhost:8081"

func log(s string, a ...interface{}) {
	fmt.Printf("%s\n", fmt.Sprintf(s, a))
}

func main() {
	fmt.Printf("listening on %s..\n", listen_on)
	local, err := net.Listen("tcp", listen_on)
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

func passData(client net.Conn, redirect net.Conn) {
    for {
        buf := make([]byte, 1024)
        n,err := client.Read(buf)
        if err != nil {
            log("passData finished (read error) %v", err)
            return
        }
        if n > 0 {
            fmt.Printf("writing %x\n", buf[0:n])
            _,werr := redirect.Write(buf[0:n])
            if werr != nil {
                log("passData finished (write error) %v", werr)
                return
            }
        }
    }
}

func handle(local net.Conn) {
	fmt.Println("Connection opened..")

    redirect,_ := net.Dial("tcp", redirect_to)
    go passData(local,redirect)
    go passData(redirect,local)
}

func fatal(s string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "fatal: %s\n", fmt.Sprintf(s, a))
	os.Exit(2)
}
