.PHONY: build start clean test

build:
	go build -o ./bin/fileservice.exe ./cmd/fileservice

start:
	nodemon --signal SIGTERM

clean:
	rm -rf ./bin ./tmp

test:
	go test ./...
