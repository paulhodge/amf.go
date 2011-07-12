include $(GOROOT)/src/Make.inc

TARG=amf

GOFILES=\
	protocol.go\
	remoting.go\
	gateway.go\

include $(GOROOT)/src/Make.pkg
