# Image configuration
IMAGE_REGISTRY ?= registry.i.jimyag.com
IMAGE_REPO ?= test
IMAGE_NAME ?= pod-validator
# Auto-generate tag with timestamp: YYYYMMDD-HH-MM-SS
# Use := to evaluate only once
IMAGE_TAG := $(shell date +%Y%m%d-%H-%M-%S)
IMAGE ?= $(IMAGE_REGISTRY)/$(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG)

# Build configuration
GOOS ?= linux
GOARCH ?= amd64

.PHONY: all build docker-build docker-push docker-build-push deploy undeploy test clean help

all: build

## Build binary
build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="-s -w" -o bin/webhook .

## Build docker image
docker-build:
	@echo "Building image: $(IMAGE)"
	docker build -t $(IMAGE) -f Dockerfile ../..

## Push docker image
docker-push:
	@echo "Pushing image: $(IMAGE)"
	docker push $(IMAGE)

## Build and push (ensures same tag is used)
docker-build-push:
	@echo "Building and pushing image: $(IMAGE)"
	docker build -t $(IMAGE) -f Dockerfile ../..
	docker push $(IMAGE)
	@echo ""
	@echo "Image: $(IMAGE)"
	@echo "Update deploy/03-deployment.yaml with this image"

## Deploy to Kubernetes
deploy:
	kubectl apply -f deploy/

## Undeploy from Kubernetes
undeploy:
	kubectl delete -f deploy/ --ignore-not-found

## Clean build artifacts
clean:
	rm -rf bin/

## Run tests against deployed webhook
test:
	@./test.sh

## Show help
help:
	@echo "Usage:"
	@echo "  make build              - Build binary"
	@echo "  make docker-build       - Build docker image"
	@echo "  make docker-push        - Push docker image"
	@echo "  make docker-build-push  - Build and push docker image"
	@echo "  make deploy             - Deploy to Kubernetes"
	@echo "  make undeploy           - Remove from Kubernetes"
	@echo "  make test               - Run tests against deployed webhook"
	@echo ""
	@echo "Variables:"
	@echo "  IMAGE_REGISTRY=$(IMAGE_REGISTRY)"
	@echo "  IMAGE_REPO=$(IMAGE_REPO)"
	@echo "  IMAGE_NAME=$(IMAGE_NAME)"
	@echo "  IMAGE_TAG=$(IMAGE_TAG)"
	@echo "  IMAGE=$(IMAGE)"
	@echo ""
	@echo "Override image tag:"
	@echo "  make docker-build IMAGE_TAG=v1.0.0"
