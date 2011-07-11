include $(GOROOT)/src/Make.inc

TARG=amv
GOFILES=\
	protocol.go\
	remoting.go\

all:
	6g protocol.go

include $(GOROOT)/src/Make.pkg
