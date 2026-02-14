.PHONY: build start clean test

build:
	go build -o ./bin/fileservice.exe ./main.go

start:
	go run main.go

clean:
	rm -rf ./bin ./tmp

test:
	go test ./...
