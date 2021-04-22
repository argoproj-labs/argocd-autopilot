VERSION=v0.1.0
OUT_DIR=dist

CLI_NAME=autopilot
IMAGE_NAMESPACE=argoproj

INSTALLATION_MANIFESTS_URL="https://raw.githubusercontent.com/argoproj/argocd-autopilot/$(VERSION)/manifests/install"
INSTALLATION_MANIFESTS_NAMESPACED_URL="https://raw.githubusercontent.com/argoproj/argocd-autopilot/$(VERSION)/manifests/namespace-install"

DEV_INSTALLATION_MANIFESTS_URL="manifests/"
DEV_INSTALLATION_MANIFESTS_NAMESPACED_URL="manifests/namespace-install"

CLI_SRCS := $(shell find . -name '*.go')

MKDOCS_DOCKER_IMAGE?=squidfunk/mkdocs-material:4.1.1
MKDOCS_RUN_ARGS?=
PACKR_CMD=$(shell if [ "`which packr`" ]; then echo "packr"; else echo "go run github.com/gobuffalo/packr/packr"; fi)

GIT_COMMIT=$(shell git rev-parse HEAD)
BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

DEV_MODE?=true

ifeq (${DEV_MODE},true)
	INSTALLATION_MANIFESTS_URL=${DEV_INSTALLATION_MANIFESTS_URL}
	INSTALLATION_MANIFESTS_NAMESPACED_URL=${DEV_INSTALLATION_MANIFESTS_NAMESPACED_URL}
endif

ifndef GOBIN
ifndef GOPATH
$(error GOPATH is not set, please make sure you set your GOPATH correctly!)
endif
GOBIN=$(GOPATH)/bin
ifndef GOBIN
$(error GOBIN is not set, please make sure you set your GOBIN correctly!)
endif
endif

define docker_build
	docker build \
		--build-arg BIN_NAME=$(1) \
		--build-arg OUT_DIR=$(OUT_DIR) \
		--build-arg BASE_ARGO_CD_URL=$(BASE_ARGO_CD_URL) \
		--build-arg BASE_ARGO_CD_NAMESPACED_URL=$(BASE_ARGO_CD_NAMESPACED_URL) \
		--build-arg BASE_APPLICATION_SET_URL=$(BASE_APPLICATION_SET_URL) \
		--target $(1) \
		-t $(IMAGE_NAMESPACE)/$(1):$(VERSION) .
endef

.PHONY: all
all: bin image

.PHONY: local
local: bin-local

.PHONY: bin
bin: cli

.PHONY: bin-local
bin-local: cli-local

.PHONY: image
image: cli-image

.PHONY: cli
cli: $(OUT_DIR)/$(CLI_NAME)-linux-amd64.gz $(OUT_DIR)/$(CLI_NAME)-linux-arm64.gz $(OUT_DIR)/$(CLI_NAME)-linux-ppc64le.gz $(OUT_DIR)/$(CLI_NAME)-linux-s390x.gz $(OUT_DIR)/$(CLI_NAME)-darwin-amd64.gz $(OUT_DIR)/$(CLI_NAME)-windows-amd64.gz

.PHONY: cli-local
cli-local: $(OUT_DIR)/$(CLI_NAME)-$(shell go env GOOS)-$(shell go env GOARCH)
	@cp $(OUT_DIR)/$(CLI_NAME)-$(shell go env GOOS)-$(shell go env GOARCH) /usr/local/bin/$(CLI_NAME)

$(OUT_DIR)/$(CLI_NAME)-linux-amd64: GO_FLAGS='GOOS=linux GOARCH=amd64 CGO_ENABLED=0'
$(OUT_DIR)/$(CLI_NAME)-darwin-amd64: GO_FLAGS='GOOS=darwin GOARCH=amd64 CGO_ENABLED=0'
$(OUT_DIR)/$(CLI_NAME)-windows-amd64: GO_FLAGS='GOOS=windows GOARCH=amd64 CGO_ENABLED=0'
$(OUT_DIR)/$(CLI_NAME)-linux-arm64: GO_FLAGS='GOOS=linux GOARCH=arm64 CGO_ENABLED=0'
$(OUT_DIR)/$(CLI_NAME)-linux-ppc64le: GO_FLAGS='GOOS=linux GOARCH=ppc64le CGO_ENABLED=0'
$(OUT_DIR)/$(CLI_NAME)-linux-s390x: GO_FLAGS='GOOS=linux GOARCH=s390x CGO_ENABLED=0'

$(OUT_DIR)/$(CLI_NAME)-%.gz: $(OUT_DIR)/$(CLI_NAME)-%
	gzip --force --keep $(OUT_DIR)/$(CLI_NAME)-$*

$(OUT_DIR)/$(CLI_NAME)-%: $(CLI_SRCS) $(GOBIN)/packr
	@ GO_FLAGS=$(GO_FLAGS) \
	BUILD_DATE=$(BUILD_DATE) \
	BINARY_NAME=$(CLI_NAME) \
	VERSION=$(VERSION) \
	GIT_COMMIT=$(GIT_COMMIT) \
	PACKR_CMD=$(PACKR_CMD) \
	OUT_FILE=$@ \
	INSTALLATION_MANIFESTS_URL=$(INSTALLATION_MANIFESTS_URL) \
	INSTALLATION_MANIFESTS_NAMESPACED_URL=$(INSTALLATION_MANIFESTS_NAMESPACED_URL) \
	MAIN=./cmd \
	./hack/build.sh

.PHONY: cli-image
cli-image: $(OUT_DIR)/$(CLI_NAME).image

$(OUT_DIR)/$(CLI_NAME).image: $(CLI_SRCS)
	$(call docker_build,$(CLI_NAME))
	@mkdir -p $(OUT_DIR)
	@touch $(OUT_DIR)/$(CLI_NAME).image

.PHONY: lint
lint: $(GOBIN)/golangci-lint
	@go mod tidy
	@echo linting go code...
	@golangci-lint run --fix --timeout 3m

.PHONY: test
test:
	./hack/test.sh

.PHONY: codegen
codegen: $(GOBIN)/mockery $(GOBIN)/interfacer
	go generate ./...

.PHONY: pre-commit
pre-commit: all lint codegen test

.PHONY: build-docs
build-docs:
	docker run ${MKDOCS_RUN_ARGS} --rm -it -p 8000:8000 -v $(shell pwd):/docs ${MKDOCS_DOCKER_IMAGE} build

.PHONY: serve-docs
serve-docs:
	docker run ${MKDOCS_RUN_ARGS} --rm -it -p 8000:8000 -v $(shell pwd):/docs ${MKDOCS_DOCKER_IMAGE} serve -a 0.0.0.0:8000

.PHONY: clean
clean:
	@rm -rf dist

$(GOBIN)/mockery:
	@curl -L -o dist/mockery.tar.gz -- https://github.com/vektra/mockery/releases/download/v1.1.1/mockery_1.1.1_$(shell uname -s)_$(shell uname -m).tar.gz
	@tar zxvf dist/mockery.tar.gz mockery
	@chmod +x mockery
	@mkdir -p $(GOBIN)
	@mv mockery $(GOBIN)/mockery
	@mockery -version

$(GOBIN)/golangci-lint:
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b `go env GOBIN) v1.36.0

$(GOBIN)/interfacer: cwd=$(shell pwd)
$(GOBIN)/interfacer:
	@cd /tmp
	GO111MODULE=on go get -v github.com/rjeczalik/interfaces/cmd/interfacer@v0.1.1
	@cd ${cwd}

$(GOBIN)/packr: cwd=$(shell pwd)
$(GOBIN)/packr:
	@cd /tmp
	GO111MODULE=on go get -v github.com/gobuffalo/packr/packr@v1.30.1
	@cd ${cwd}
