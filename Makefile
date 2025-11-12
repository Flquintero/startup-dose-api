.PHONY: run build test clean docker-build docker-run

run:
	go run ./cmd/server

build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

test:
	go test -v ./...

clean:
	rm -rf bin/
	go clean

docker-build:
	docker build -t go-proxy:latest .

docker-run: docker-build
	docker run -p 8080:8080 go-proxy:latest

help:
	@echo "Available targets:"
	@echo "  make run          - Run the server"
	@echo "  make build        - Build the binary"
	@echo "  make test         - Run tests"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-run   - Build and run Docker container"
