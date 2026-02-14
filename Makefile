.PHONY: build start docker-build docker-run test docker-push clean

build:
	go build -o ./bin/app main.go

start:
	nodemon --signal SIGTERM

# Docker targets
docker-build:
	docker build -t file-service:local .

docker-run:
	docker run --rm -p 8080:8080 --env-file .env file-service:local

# Testing
test:
	go vet ./...
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf ./bin/
