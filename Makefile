.PHONY: run build test clean docker-build docker-run login tag push all release

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
	docker build -t $(IMAGE_NAME):$(TAG) .

docker-run: docker-build
	docker run -p 8080:8080 go-proxy:latest

# ECR Configuration
ECR_REGISTRY = 799771184733.dkr.ecr.us-east-2.amazonaws.com
ECR_REPO     = startup-dose/api
IMAGE_NAME   = startup-dose-api
TAG         ?= latest
REGION      = us-east-2

# ECR Targets
login:
	@echo "Logging into ECR..."
	aws ecr get-login-password --region $(REGION) | docker login --username AWS --password-stdin $(ECR_REGISTRY)

build: docker-build

tag:
	@echo "Tagging image $(IMAGE_NAME):$(TAG) as $(ECR_REGISTRY)/$(ECR_REPO):$(TAG)"
	docker tag $(IMAGE_NAME):$(TAG) $(ECR_REGISTRY)/$(ECR_REPO):$(TAG)

push: tag
	@echo "Pushing image to ECR: $(ECR_REGISTRY)/$(ECR_REPO):$(TAG)"
	docker push $(ECR_REGISTRY)/$(ECR_REPO):$(TAG)

all: login build tag push

release: all

help:
	@echo "Available targets:"
	@echo "  make run          - Run the server"
	@echo "  make build        - Build the binary"
	@echo "  make test         - Run tests"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-run   - Build and run Docker container"
	@echo ""
	@echo "ECR targets:"
	@echo "  make login        - Login to ECR"
	@echo "  make build        - Build Docker image"
	@echo "  make tag          - Tag image for ECR (TAG=v0.1.0)"
	@echo "  make push         - Push image to ECR"
	@echo "  make all          - Login, build, tag, and push (TAG=v0.1.0)"
	@echo "  make release      - Alias for 'make all'"
