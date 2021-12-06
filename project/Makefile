test:
	go test ./...

build:
	go build -o bin/main main.go

lint:
	golangci-lint run

all: test lint build