all: build run

build:
	6g test.go && 6l test.6
run:
	./6.out
