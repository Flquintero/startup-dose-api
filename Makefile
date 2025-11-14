.PHONY: run build-local test clean docker-build docker-run login tag push all release cert-request cert-check cert-list deploy-stack deploy dns-instructions dns-check

run:
	go run ./cmd/server

build-local:
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

tag:
	@echo "Tagging image $(IMAGE_NAME):$(TAG) as $(ECR_REGISTRY)/$(ECR_REPO):$(TAG)"
	docker tag $(IMAGE_NAME):$(TAG) $(ECR_REGISTRY)/$(ECR_REPO):$(TAG)

push: tag
	@echo "Pushing image to ECR: $(ECR_REGISTRY)/$(ECR_REPO):$(TAG)"
	docker push $(ECR_REGISTRY)/$(ECR_REPO):$(TAG)

all: login docker-build tag push

release: all

# Deployment Configuration
DOMAIN_NAME      = api.startupdose.com
STACK_NAME      ?= startup-dose-api-production
CERTIFICATE_ARN ?=
VPC_ID          ?=
SUBNET_IDS      ?=

# Certificate Management
cert-request:
	@./scripts/setup-certificate.sh --domain $(DOMAIN_NAME) --region $(REGION)

cert-check:
ifndef CERTIFICATE_ARN
	@echo "Error: CERTIFICATE_ARN is required"
	@echo "Usage: make cert-check CERTIFICATE_ARN=arn:aws:acm:..."
	@exit 1
endif
	@./scripts/setup-certificate.sh --certificate-arn $(CERTIFICATE_ARN) --region $(REGION)

cert-list:
	@./scripts/setup-certificate.sh --list --region $(REGION)

# CloudFormation Deployment
deploy-stack:
ifndef VPC_ID
	@echo "Error: VPC_ID is required"
	@echo "Usage: make deploy-stack VPC_ID=vpc-xxx SUBNET_IDS=subnet-xxx,subnet-yyy CERTIFICATE_ARN=arn:..."
	@exit 1
endif
ifndef SUBNET_IDS
	@echo "Error: SUBNET_IDS is required (comma-separated)"
	@echo "Usage: make deploy-stack VPC_ID=vpc-xxx SUBNET_IDS=subnet-xxx,subnet-yyy CERTIFICATE_ARN=arn:..."
	@exit 1
endif
	@./scripts/deploy-cloudformation.sh \
		--stack-name $(STACK_NAME) \
		--vpc-id $(VPC_ID) \
		--subnet-ids $(SUBNET_IDS) \
		$(if $(CERTIFICATE_ARN),--certificate-arn $(CERTIFICATE_ARN),--skip-certificate) \
		--image-tag $(TAG)

# Full deployment pipeline
deploy: all deploy-stack
	@echo "Deployment complete!"

# DNS Setup targets
dns-instructions:
	@./scripts/setup-dns.sh

dns-check:
	@./scripts/setup-dns.sh --check

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
	@echo ""
	@echo "Certificate targets:"
	@echo "  make cert-request                    - Request SSL certificate for api.startupdose.com"
	@echo "  make cert-check CERTIFICATE_ARN=...  - Check certificate validation status"
	@echo "  make cert-list                       - List all certificates"
	@echo ""
	@echo "Deployment targets:"
	@echo "  make deploy-stack VPC_ID=... SUBNET_IDS=... [CERTIFICATE_ARN=...] - Deploy CloudFormation stack"
	@echo "  make deploy VPC_ID=... SUBNET_IDS=... CERTIFICATE_ARN=...         - Build, push, and deploy"
	@echo ""
	@echo "DNS Setup targets:"
	@echo "  make dns-instructions  - Show GoDaddy DNS setup instructions"
	@echo "  make dns-check         - Check if DNS is configured correctly"
	@echo ""
	@echo "Example workflow:"
	@echo "  1. make cert-request"
	@echo "  2. Add DNS validation CNAME to GoDaddy, wait for validation"
	@echo "  3. make cert-check CERTIFICATE_ARN=arn:..."
	@echo "  4. make deploy VPC_ID=vpc-xxx SUBNET_IDS=subnet-x,subnet-y CERTIFICATE_ARN=arn:..."
	@echo "  5. make dns-instructions (get ALB DNS and add application CNAME to GoDaddy)"
	@echo "  6. make dns-check (verify DNS is working)"
