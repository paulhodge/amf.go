package amfgo

import (
        "bufio"
        "bytes"
        "encoding/binary"
        "fmt"
        "io/ioutil"
        "os"
        "http"
        )

const _AMF0 = 0
const _AMF3 = 0

// Interface for the local Reader. This is implemented by Reader
type Reader interface {
    Read(p []byte) (n int, err os.Error)
    Peek(n int) ([]byte, os.Error)
}

type Page struct {
    Title string
    Body []byte
}

func (p *Page) save() os.Error {
    filename := p.Title + ".txt"
    return ioutil.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, os.Error) {
    filename := title + ".txt"
    body, err := ioutil.ReadFile(filename)
    if err != nil {
        fmt.Printf("File not found: %s", filename)
        return nil, err
    }
    return &Page{Title: title, Body: body}, nil
}

const SERVER_NAME = "SERVER_NAME"

type Gateway struct {
}

type DecodeError struct {
    msg string
}

type DecodeResult struct {
    headers map[string]string
    body map[string]string
}

type DecodeContext struct {
    amfVersion uint16
    useAmf3 bool
}

// Helper functions.
func readByte(stream Reader) (uint8, os.Error) {
    buf := make([]byte, 1)
    _, err := stream.Read(buf)
    return buf[0], err
}

func readUint8(stream Reader) uint8 {
    var value uint8
    binary.Read(stream, binary.BigEndian, &value)
    return value
}
func readUint16(stream Reader) uint16 {
    var value uint16
    binary.Read(stream, binary.BigEndian, &value)
    return value
}
func readUint32(stream Reader) uint32 {
    var value uint32
    binary.Read(stream, binary.BigEndian, &value)
    return value
}
func readFloat64(stream Reader) float64 {
    var value float64
    binary.Read(stream, binary.BigEndian, &value)
    return value
}
func readString(stream Reader) string {
    length := readUint16(stream)
    data := make([]byte, length)
    stream.Read(data)
    return string(data)
}
func readStringLength(stream Reader, length int) string {
    data := make([]byte, length)
    stream.Read(data)
    return string(data)
}
func peekByte(stream Reader) uint8 {
    buf, _ := stream.Peek(1)
    return buf[0]
}

func unused(... interface{}) {
}

func decode(stream Reader) (*DecodeResult, *DecodeError) {

    cxt := DecodeContext{}
    result := DecodeResult{}

    cxt.amfVersion = readUint16(stream)

    fmt.Printf("amfVersion = %d\n", cxt.amfVersion)

    // see http://osflash.org/documentation/amf/envelopes/remoting#preamble
    // why we are doing this...
    if cxt.amfVersion > 0x09 {
        return nil, &DecodeError{"Malformed stream (wrong amfVersion)"}
    }

    cxt.useAmf3 = cxt.amfVersion == _AMF3
    headerCount := readUint16(stream)

    fmt.Printf("headerCount = %d\n", headerCount)

    // Read headers
    for i := 0; i < int(headerCount); i++ {
        name := readString(stream)
        required := readUint8(stream)
        messageLength := readUint32(stream)

        // Check for AMF3 type marker
        if (cxt.amfVersion == _AMF3) {
            // TODO
        }

        value := readObject(stream, &cxt)

        unused(name, required, messageLength, value)

        fmt.Printf("Read header, name = %s", name)
    }

    // Read bodies
    messageCount := readUint16(stream)
    fmt.Printf("messageCount = %d\n", messageCount)

    for i := 0; i < int(messageCount); i++ {
        targetUri := readString(stream)
        responseUri := readString(stream)
        messageLength := readUint32(stream)

        unused(targetUri, responseUri, messageLength)

        fmt.Printf("Read body, targetUri = %s, messageLength = %d\n", targetUri, messageLength)
    }

    return &result, nil
}

func readElement(stream Reader, cxt *DecodeContext) {
    var typeCode uint8
    binary.Read(stream, binary.BigEndian, &typeCode)
}

func readObject(stream Reader, cxt *DecodeContext) interface{} {
    if cxt.amfVersion == 0 {
        return readAMF0Object(stream)
    }

    return readAMF3Object(stream)
}

func readAMF0Object(stream Reader) interface{} {
    typeMarker,_ := readByte(stream)

    // Type markers
    const (
        numberType = 0
        booleanType = 1
        stringType = 2
        objectType = 3
        movieClipType = 4
        nullType = 5
        undefinedType = 6
        referenceType = 7
        ecmaArrayType = 8
        objectEndType = 9
        strictArrayType = 10
        dateType = 11
        longStringType = 12
        unsupporedType = 13
        recordsetType = 14
        xmlObjectType = 15
        typedObjectType = 16
        avmPlusObjectType = 17
    )

    switch typeMarker {
    case numberType:
        return readFloat64(stream)
    case booleanType:
        return readUint8(stream) != 0
    case stringType:
        return readString(stream)
    case objectType:
        result := map[string]interface{} {}
        for true {
            c1,_ := readByte(stream)
            c2,_ := readByte(stream)
            length := int(c1) << 8 + int(c2)
            name := readStringLength(stream, length)
            k := peekByte(stream)
            if k == 0x09 {
                break
            }
            result[name] = readAMF0Object(stream)
        }
        return result

    case movieClipType:
        fmt.Printf("movie clip type not supported")
    case nullType:
        return nil
    case undefinedType:
    case referenceType:
    case ecmaArrayType:
    case objectEndType:
    case strictArrayType:
    case dateType:
    case longStringType:
    case unsupporedType:
    case recordsetType:
    case xmlObjectType:
    case typedObjectType:
    case avmPlusObjectType:
        return readAMF3Object(stream)
    }

    fmt.Printf("AMF0 type marker was not supported: %d", typeMarker)
    return nil
}

func readAMF3Object(stream Reader) interface{} {
    typeMarker,_ := readByte(stream)

    // Type markers
    const (
        undefinedType = 0
        nullType = 1
        falseType = 2
        trueType = 3
        integerType = 4
        doubleType = 5
        stringType = 6
        xmlType = 7
        dateType = 8
        arrayType = 9
        objectType = 10
        avmPlusXmlType = 11
        byteArrayType = 12
    )

    fmt.Printf("AMF3 type marker was not supported: %d", typeMarker)
    return nil
}

// Read a 29-bit compact encoded integer.
func readInt29(stream Reader) (uint32, os.Error) {
    var result uint32 = 0
    for i := 0; i < 4; i++ {
        _byte, err := readByte(stream)

        if err != nil {
            return result, err
        }

        result = result << 8
        result += uint32(_byte)

        if (_byte & 0x80) == 0 {
            break
        }
    }
    return result, nil
}

func test_readInt29(data []byte) {
    orig := data
    result,_ := readInt29(bufio.NewReader(bytes.NewBuffer(data)))
    fmt.Printf("unpacked %x to %d\n", orig, result)
}

func handleGet(w http.ResponseWriter) {
    w.Header().Set("Content-Type", "text/plain")
    w.WriteHeader(405)
    fmt.Fprintf(w, "405 Method Not Allowed\n\n"+
            "To access this amfgo gateway you must use POST requests "+
            "(%s received))")
}

func writeReply500(w http.ResponseWriter) {
    w.Header().Set("Content-Type", "text/plain")
    w.WriteHeader(500)
    fmt.Fprintf(w, "500 Internal Server Error\n\n"+
            "Unexplained error")
}

func handler(w http.ResponseWriter, r *http.Request) {
    if r.Method == "Get" {
        handleGet(w)
        return
    }

    // Decode the request
    /*result,error :=*/ decode(bufio.NewReader(r.Body))

    // remoting.decode

    // process request (to callback)

    // remoting.encode

    writeReply500(w)

}
func main() {
    //test_readInt29([]byte{0x7f})
    //test_readInt29([]byte{0xff,0x7f})
    //test_readInt29([]byte{0xff,0xff,0x7f})

    http.HandleFunc("/", handler)
    http.ListenAndServe(":8080", nil)
}
