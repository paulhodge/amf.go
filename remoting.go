package amf

import (
	"bytes"
	"fmt"
	"http"
	"io"
	"os"
	"strconv"
)

import amf "./protocol"

// function for WIP code:
func unused(a ...interface{}) {}

type FlexAbstractMessage struct {
	Body        []byte
	ClientId    string
	Destination string
	Headers     map[string]interface{}
	MessageId   string
	Timestamp   uint32
	TimeToLive  uint32
}

type FlexAsyncMessage struct {
	AbstractMessage FlexAbstractMessage
	CorrelationId   string
}

type FlexCommandMessage struct {
	AsyncMessage FlexAsyncMessage
	Operation    uint
}

type FlexRemotingMessage struct {
    // AbstractMessage:
	Body        []interface{}
	ClientId    string
	Destination string
	Headers     map[string]interface{}
	MessageId   string
	Timestamp   uint32
	TimeToLive  uint32

	Operation string
	Source    string
}

type FlexAcknowledgeMessage struct {
	AsyncMessage FlexAsyncMessage
	flags        byte
}

type FlexErrorMessage struct {
	AcknowledgeMessage FlexAcknowledgeMessage
	ExtendedData       string
	FaultCode          int
	FaultDetail        int
	FaultString        string
	RootCause          string
}

type MessageBundle struct {
	AmfVersion uint16
	Headers    []AmfHeader
	Messages   []AmfMessage
}

type AmfHeader struct {
	Name           string
	MustUnderstand bool
	Value          interface{}
}
type AmfMessage struct {
	TargetUri   string
	ResponseUri string
	Body        interface{}
}


func readRequestArgs(stream io.Reader, cxt *amf.Decoder) []interface{} {
	/*
	   lookaheadByte := peekByte(stream)
	   if lookaheadByte == 17 {
	       if !cxt.useAmf3() {
	           fmt.Printf("Unexpected AMF3 type with incorrect message type")
	       }
	       fmt.Printf("while reading args, found next byte of 17")
	       return nil
	   }

	   if lookaheadByte != 10 {
	       fmt.Printf("Strict array type required for request body (found %d)", lookaheadByte)
	       return nil
	   }
	*/

	cxt.ReadByte()

	count := cxt.ReadUint32()
	result := make([]interface{}, count)

	fmt.Printf("argument count = %d\n", count)

	for i := uint32(0); i < count; i++ {
		result[i] = cxt.ReadValue()
		fmt.Printf("parsed value %s", result[i])
	}
	return result
}

func DecodeMessageBundle(stream io.Reader) (*MessageBundle, os.Error) {

	cxt := amf.NewDecoder(stream, 0)
    cxt.RegisterType("flex.messaging.messages.RemotingMessage", FlexRemotingMessage{})

	amfVersion := cxt.ReadUint16()

	result := MessageBundle{}
	cxt.AmfVersion = amfVersion
	result.AmfVersion = amfVersion

	/*
	   From http://osflash.org/documentation/amf/envelopes/remoting:

	   The first two bytes of an AMF message are an unsigned short int. The result 
	   indicates what type of Flash Player connected to the server.

	   0x00 for Flash Player 8 and below
	   0x01 for FlashCom/FMS
	   0x03 for Flash Player 9
	   Note that Flash Player 9 will always set the second byte to 0x03, regardless of
	   whether the message was sent in AMF0 or AMF3.
	*/

	if cxt.AmfVersion > 0x09 {
		return nil, os.NewError("Malformed stream (wrong amfVersion)")
	}

	headerCount := cxt.ReadUint16()

	/*
	   From http://osflash.org/documentation/amf/envelopes/remoting:

	   Each header consists of the following:

	   UTF string (including length bytes) - name
	   Boolean - specifies if understanding the header is 'required'
	   Long - Length in bytes of header
	   Variable - Actual data (including a type code)
	*/

	// Read headers
	result.Headers = make([]AmfHeader, headerCount)
	for i := 0; i < int(headerCount); i++ {
		name := cxt.ReadString()
		mustUnderstand := cxt.ReadUint8() != 0
		messageLength := cxt.ReadUint32()
		unused(messageLength)

		// TODO: Check for AMF3 type marker?

		value := cxt.ReadValue()
		header := AmfHeader{name, mustUnderstand, value}
		result.Headers[i] = header

		fmt.Printf("Read header, name = %s", name)
	}

	/*
	   From http://osflash.org/documentation/amf/envelopes/remoting:

	   Between the headers and the start of the bodies is a int specifying the number of
	   bodies. Each body consists of the following:

	   UTF String - Target
	   UTF String - Response
	   Long - Body length in bytes
	   Variable - Actual data (including a type code)
	*/

	// Read message bodies
	messageCount := cxt.ReadUint16()
	result.Messages = make([]AmfMessage, messageCount)

	for i := 0; i < int(messageCount); i++ {
		// TODO: Should reset object tables here

		message := &result.Messages[i]

		message.TargetUri = cxt.ReadString()
		message.ResponseUri = cxt.ReadString()

		messageLength := cxt.ReadUint32()

        is_request := true

        // TODO: Check targetUri to see if this isn't an array?

		if is_request {
            // Read an array, however this array is strange because it doesn't use
            // the reference bit.
            typeCode := cxt.ReadUint8()
            unused(typeCode)
            ref := cxt.ReadUint32()
            itemCount := int(ref)
            args := make([]interface{}, itemCount)
            for i := 0; i < itemCount; i++ {
                args[i] = cxt.ReadValue()
            }
            message.Body = args
		} else {
            message.Body = cxt.ReadValue()
        }

		unused(messageLength)
	}

	return &result, nil
}

func encodeBundle(cxt *amf.Encoder, bundle *MessageBundle) os.Error {
	cxt.WriteUint16(bundle.AmfVersion)

	// Write headers
	cxt.WriteUint16(uint16(len(bundle.Headers)))
	for _, header := range bundle.Headers {
		cxt.WriteString(header.Name)
		cxt.WriteBool(header.MustUnderstand)
	}

	// Write messages
	cxt.WriteUint16(uint16(len(bundle.Messages)))
	for _, message := range bundle.Messages {
		cxt.WriteString(message.TargetUri)
		cxt.WriteString(message.ResponseUri)
		cxt.WriteUint32(0)

		cxt.WriteValueAmf3(message.Body)
	}

	return nil
}

// This gateway stuff will get moved to a separate file..

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


func HttpHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "Get" {
		handleGet(w)
		return
	}

    bits := make([]byte, 3024)
    n,_ := r.Body.Read(bits)
    fmt.Printf("body = %x\n", bits[0:n])

	// Decode the request
	requestBundle, _ := DecodeMessageBundle(r.Body)

	// Initialize the reply bundle.
	replyBundle := MessageBundle{}
	replyBundle.AmfVersion = 3
	replyBundle.Messages = make([]AmfMessage, len(requestBundle.Messages))

	// Construct a reply to each message.
	for index, request := range requestBundle.Messages {
		reply := &replyBundle.Messages[index]

		replyBody, success := amfMessageHandler(request)
		reply.Body = replyBody

		/*
		   From http://osflash.org/documentation/amf/envelopes/remoting:

		   The response to a request has the exact same structure as a request. A request
		   requiring a body response should be answered in the following way:

		   Target: set to Response index plus one of "/onStatus", "onResult", or
		   "/onDebugEvents". "/onStatus" is reserved for runtime errors. "/onResult" is for
		   succesful calls. "/onDebugEvents" is for debug information, see debug information.
		   Thus if the client requested something with response index '/1', and the call was
		   succesful, '/1/onResult' should be sent back. Response: should be set to the string
		   'null'.  Data: set to the returned data.
		*/

		if success {
			reply.TargetUri = request.TargetUri + "/onResult"
		} else {
			reply.TargetUri = request.TargetUri + "/onStatus"
		}
		reply.ResponseUri = ""
		fmt.Printf("writing reply to message %d, targetUri = %s", index, reply.TargetUri)
	}

	// Encode the outgoing message bundle.
	replyBuffer := bytes.NewBuffer(make([]byte, 0))
	encoder := amf.NewEncoder(replyBuffer)
	encodeBundle(encoder, &replyBundle)
	replyBytes := replyBuffer.Bytes()
	w.Write(replyBytes)

	w.Header().Set("Content-Type", "application/x-amf")
	w.Header().Set("Content-Length", strconv.Itoa(len(replyBytes)))
	w.Header().Set("Server", "SERVER_NAME")

	fmt.Printf("writing reply data with length: %d", len(replyBytes))
}

func amfMessageHandler(request AmfMessage) (data interface{}, success bool) {
	return "hello", true
}

func ServeHttp() {
	http.HandleFunc("/", HttpHandler)
	http.ListenAndServe(":8082", nil)
}
