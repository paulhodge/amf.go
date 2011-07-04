all: build run

build:
	6g protocol.go && 6l protocol.6
run:
	./6.out
