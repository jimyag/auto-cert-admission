# Image configuration
IMAGE_REGISTRY ?= docker.io
IMAGE_REPO ?= jimyag
IMAGE_NAME ?= pod-label-injector
IMAGE_TAG ?= latest
IMAGE ?= $(IMAGE_REGISTRY)/$(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG)

# Build configuration
GOOS ?= linux
GOARCH ?= amd64

.PHONY: all build docker-build docker-push deploy undeploy clean

all: build

## Build binary
build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="-s -w" -o bin/webhook .

## Build docker image
docker-build:
	docker build -t $(IMAGE) -f Dockerfile ../..

## Push docker image
docker-push:
	docker push $(IMAGE)

## Build and push
docker-build-push: docker-build docker-push

## Deploy to Kubernetes
deploy:
	kubectl apply -f deploy/

## Undeploy from Kubernetes
undeploy:
	kubectl delete -f deploy/ --ignore-not-found

## Clean build artifacts
clean:
	rm -rf bin/

## Show help
help:
	@echo "Usage:"
	@echo "  make build              - Build binary"
	@echo "  make docker-build       - Build docker image"
	@echo "  make docker-push        - Push docker image"
	@echo "  make docker-build-push  - Build and push docker image"
	@echo "  make deploy             - Deploy to Kubernetes"
	@echo "  make undeploy           - Remove from Kubernetes"
	@echo ""
	@echo "Variables:"
	@echo "  IMAGE_REGISTRY=$(IMAGE_REGISTRY)"
	@echo "  IMAGE_REPO=$(IMAGE_REPO)"
	@echo "  IMAGE_NAME=$(IMAGE_NAME)"
	@echo "  IMAGE_TAG=$(IMAGE_TAG)"
	@echo "  IMAGE=$(IMAGE)"
