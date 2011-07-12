include $(GOROOT)/src/Make.inc

TARG=amf

GOFILES=\
	protocol.go\
	remoting.go\

include $(GOROOT)/src/Make.pkg
