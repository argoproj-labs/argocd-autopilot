

OUT_DIR="./dist"
BINARY_NAME="cf-argo"

VERSION="v0.0.2"
GIT_COMMIT=$(shell git rev-parse HEAD)

BASE_GIT_URL="https://github.com/codefresh-io/argocd-template@v0.1.0"

ifndef GOPATH
$(error GOPATH is not set, please make sure you set your GOPATH correctly!)
endif

.PHONY: build
build:
	@ OUT_DIR=$(OUT_DIR) \
	BINARY_NAME=$(BINARY_NAME) \
	VERSION=$(VERSION) \
	GIT_COMMIT=$(GIT_COMMIT) \
	BASE_GIT_URL=$(BASE_GIT_URL) \
	./hack/build.sh

.PHONY: install
install: build
	@rm /usr/local/bin/$(BINARY_NAME) || true
	@ln -s $(shell pwd)/$(OUT_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

$(GOPATH)/bin/golangci-lint:
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b `go env GOPATH`/bin v1.33.2

.PHONY: lint
lint: $(GOPATH)/bin/golangci-lint
	@go mod tidy
	@echo linting go code...
	@golangci-lint run --fix --timeout 3m

.PHONY: clean
clean:
	@rm -rf dist
