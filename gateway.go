package amf

import (
    "bytes"
    "fmt"
    "http"
    "strconv"
)

func handleGet(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(405)
	fmt.Fprintf(w, "405 Method Not Allowed\n\n"+
		"To access this amf.go gateway you must use POST requests "+
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
	encoder := NewEncoder(replyBuffer)
	EncodeMessageBundle(encoder, &replyBundle)
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
