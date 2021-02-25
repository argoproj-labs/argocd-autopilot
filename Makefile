

OUT_DIR="./dist"
BINARY_NAME="cf-argo"

VERSION="v0.0.2"
GIT_COMMIT=$(shell git rev-parse HEAD)

BASE_GIT_URL="https://github.com/codefresh-io/argocd-template@v0.1.2"

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

.PHONY: lint
lint: $(GOPATH)/bin/golangci-lint
	@go mod tidy
	@echo linting go code...
	@golangci-lint run --fix --timeout 3m

.PHONY: test
test:
	./hack/test.sh

.PHONY: codegen
codegen: $(GOPATH)/bin/mockery
	go generate ./pkg/git

.PHONY: pre-commit
pre-commit: lint build codegen test

.PHONY: clean
clean:
	@rm -rf dist

$(GOPATH)/bin/mockery:
	@curl -L -o dist/mockery.tar.gz -- https://github.com/vektra/mockery/releases/download/v1.1.1/mockery_1.1.1_$(shell uname -s)_$(shell uname -m).tar.gz
	@tar zxvf dist/mockery.tar.gz mockery
	@chmod +x mockery
	@mkdir -p $(GOPATH)/bin
	@mv mockery $(GOPATH)/bin/mockery
	@mockery -version

$(GOPATH)/bin/golangci-lint:
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b `go env GOPATH`/bin v1.36.0

