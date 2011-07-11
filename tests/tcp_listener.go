package main

import (
	"net"
	"fmt"
	"io"
	"os"
)

import amf "../protocol"

const listen_on = ":8081"

func main() {
	fmt.Printf("listening on %s\n", listen_on)
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

type ReaderSpy struct {
	reader io.Reader
	data   []byte
}

func (r *ReaderSpy) Read(p []byte) (int, os.Error) {
	n, err := r.reader.Read(p)
	r.data = append(r.data, p...)
	return n, err
}

func handle(local net.Conn) {
	fmt.Println("Connection opened..")

	// Socket.mxml will send a bunch of AMF3 encoded values, each preceded by
	// a string label.

	//outgoing := bytes.NewBuffer([]byte{})
    cxt := amf.NewDecoder(local, 3)

	for {
		label := cxt.ReadString()
		if label == "" {
			fmt.Println("Received empty label")
			break
		}
        fmt.Printf("Received label: %s\n", label)

		cxt.ReadValueAmf3()
/*
		// Spy on the data that was read.
		readerSpy := ReaderSpy{}
		cxt.stream = local

		obj, err := amf.ReadValueAmf3(&readerSpy)
		if err != nil {
			fmt.Printf("%v\n", err)
			break
		}
		fmt.Printf("%s %x -> %v\n", label, readerSpy.data, obj)

		// Write the value to our outgoing buffer.
		amf.WriteString(outgoing, label)
		amf.WriteValueAmf3(outgoing, obj)
        */
	}

/*
	// Write all of our data, prepended with size.
	outgoingData := outgoing.Bytes()
	amf.WriteInt32(local, int32(len(outgoingData)))
	fmt.Printf("sending %d bytes\n", len(outgoingData))

	local.Write(outgoingData)
	amf.WriteString(local, "")
    */
}

func log(s string, a ...interface{}) {
	fmt.Printf("%s\n", fmt.Sprintf(s, a))
}
func fatal(s string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "fatal: %s\n", fmt.Sprintf(s, a))
	os.Exit(2)
}
