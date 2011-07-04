all: build run

build:
	6g decode.go && 6l decode.6
run:
	./6.out
