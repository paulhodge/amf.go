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

// For in-progress code:
func unused(... interface{}) { }

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
    headers []AmfHeader
    body map[string]string
}

type DecodeContext struct {
    amfVersion uint16
    useAmf3 bool

    stringTable []string
    classTable []AvmClass
    objectTable []AvmObject
}

type AmfHeader struct {
    name string
    mustUnderstand bool
    value interface{}
}

type AvmObject struct {
    class *AvmClass
}

type AvmClass struct {
    name string
    externalizable bool
    dynamic bool
    properties []string
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

// Read a 29-bit compact encoded integer (as defined in AVM3)
func readUint29(stream Reader) (uint32, os.Error) {
    var result uint32 = 0
    for i := 0; i < 4; i++ {
        b, err := readByte(stream)

        if err != nil {
            return result, err
        }

        result = (result << 7) + (uint32(b) & 0x7f)

        if (b & 0x80) == 0 {
            break
        }
    }
    return result, nil
}

func readStringAmf3(stream Reader, cxt *DecodeContext) (string, os.Error) {
    ref,_ := readUint29(stream)

    // Check the low bit to see if this is a reference
    if (ref & 1) == 0 {
        return cxt.stringTable[int(ref>>1)],nil
    }

    length := int(ref >> 1)

    if (length == 0) {
        return "", nil
    }

    str := readStringLength(stream, length)

    cxt.stringTable = append(cxt.stringTable, str)

    return str, nil
}

func readObjectAmf3(stream Reader, cxt *DecodeContext) (*AvmObject, os.Error) {

    ref,_ := readUint29(stream)

    fmt.Printf("in readObjectAmf3, parsed ref: %d\n", ref)

    // Check the low bit to see if this is a reference
    if (ref & 1) == 0 {
        return &cxt.objectTable[int(ref >> 1)], nil
    }

    class := readClassDefinitionAmf3(stream, ref, cxt)

    return &result,nil
}

func readClassDefinitionAmf3(stream Reader, ref uint32, cxt *DecodeContext) *AmvClass {

    // Check for a reference to an existing class definition
    if (ref & 2) == 0 {
        return &cxt.objectTable[int(ref >> 2)], nil
    }

    // Parse a class definition
    className,_ := readStringAmf3(stream, cxt)
    fmt.Printf("read className = %s\n", className)

    externalizable := ref & 4 != 0
    dynamic := ref & 8 != 0
    propertyCount := ref >> 4

    unused(externalizable, dynamic)

    fmt.Printf("read propertyCount = %d\n", propertyCount)

    class := AvmClass{className, externalizable, dynamic, make([]string, propertyCount)}

    // Property names
    for i := 0; i < propertyCount; i++ {
        class.properties[i] = readStringAmf3(stream, cxt)
    }

    return &class
}

func readArrayAmf3(stream Reader, cxt *DecodeContext) (interface{}, os.Error) {
    ref,_ := readUint29(stream)

    fmt.Printf("readArrayAmf3 read ref: %d\n", ref)

    // Check the low bit to see if this is a reference
    if (ref & 1) == 0 {
        return &cxt.objectTable[int(ref >> 1)], nil
    }

    size := int(ref >> 1)

    fmt.Printf("readArrayAmf3 read size: %d", size)

    key,_ := readStringAmf3(stream, cxt)

    if key == "" {
        // No key, the whole array is dense.
        result := make([]interface{}, size)

        for i := 0; i < size; i++ {
            result[size] = readValueAmf3(stream, cxt)
        }
        return result, nil
    }

    // There are keys, return a mixed array.


    unused(size)

    return nil,nil

}

func peekByte(stream Reader) uint8 {
    buf, _ := stream.Peek(1)
    return buf[0]
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
    result.headers = make([]AmfHeader, headerCount)
    for i := 0; i < int(headerCount); i++ {
        name := readString(stream)
        mustUnderstand := readUint8(stream) != 0
        messageLength := readUint32(stream)
        unused(messageLength)

        // Check for AMF3 type marker
        if (cxt.amfVersion == _AMF3) {
            typeMarker := peekByte(stream)
            if typeMarker == 17 {
                fmt.Printf("found AMF3 type marker on header")
            }
        }

        value := readValue(stream, &cxt)
        header := AmfHeader{name, mustUnderstand, value}
        result.headers[i] = header

        fmt.Printf("Read header, name = %s", name)
    }

    // Read bodies
    messageCount := readUint16(stream)
    fmt.Printf("messageCount = %d\n", messageCount)

    for i := 0; i < int(messageCount); i++ {

        targetUri := readString(stream)
        responseUri := readString(stream)
        messageLength := readUint32(stream)

        fmt.Printf("Read targetUri = %s\n", targetUri)
        fmt.Printf("Read responseUri = %s\n", responseUri)
        fmt.Printf("Read messageLength = %d\n", messageLength)

        is_request := true
        if is_request {
            readRequestArgs(stream, &cxt)
        }

        messageBody := readValue(stream, &cxt)

        unused(targetUri, responseUri, messageLength, messageBody)

    }

    return &result, nil
}

func readRequestArgs(stream Reader, cxt *DecodeContext) []interface{} {
    lookaheadByte := peekByte(stream)
    if lookaheadByte == 17 {
        if !cxt.useAmf3 {
            fmt.Printf("Unexpected AMF3 type with incorrect message type")
        }
        fmt.Printf("while reading args, found next byte of \\x11")
        return nil
    }

    if lookaheadByte != 10 {
        fmt.Printf("Strict array type required for request body (found %d)", lookaheadByte)
        return nil
    }

    readByte(stream)

    count := readUint32(stream)
    result := make([]interface{}, 0)

    fmt.Printf("argument count = %d\n", count)

    for i := uint32(0); i < count; i++ {
        result[i] = readValue(stream, cxt)
    }
    return result
}

func readValue(stream Reader, cxt *DecodeContext) interface{} {
    if cxt.amfVersion == 0 {
        return readValueAmf0(stream, cxt)
    }

    return readValueAmf3(stream, cxt)
}

func readValueAmf0(stream Reader, cxt *DecodeContext) interface{} {
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
            result[name] = readValueAmf0(stream, cxt)
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
        return readValueAmf3(stream, cxt)
    }

    fmt.Printf("AMF0 type marker was not supported: %d", typeMarker)
    return nil
}

func readValueAmf3(stream Reader, cxt *DecodeContext) interface{} {
    typeMarker,_ := readByte(stream)

    if typeMarker == 17 {
        typeMarker,_ = readByte(stream)
    }

    fmt.Printf("read typeMarker: %d\n", typeMarker)

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

    switch typeMarker {
    case nullType:
        return nil
    case falseType:
        return false
    case trueType:
        return true
    case integerType:
        result,_ := readUint29(stream)
        return result
    case doubleType:
        return readFloat64(stream)
    case stringType:
        result,_ := readStringAmf3(stream, cxt)
        return result
    case objectType:
        result,_ := readObjectAmf3(stream, cxt)
        return result
    case arrayType:
        result,_ := readArrayAmf3(stream, cxt)
        return result
    }

    fmt.Printf("AMF3 type marker was not supported: %d\n", typeMarker)
    return nil
}


func test_readInt29(data []byte) {
    orig := data
    result,_ := readUint29(bufio.NewReader(bytes.NewBuffer(data)))
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
