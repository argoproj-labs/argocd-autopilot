VERSION=v0.0.3
OUT_DIR=dist

CLI_NAME=autopilot
AGENT_NAME=gitops-agent
IMAGE_NAMESPACE=codefresh-io

CLI_PKGS := $(shell echo cmd/$(CLI_NAME) && go list -f '{{ join .Deps "\n" }}' ./cmd/$(CLI_NAME)/ | grep 'github.com/argoproj/argocd-autopilot/' | cut -c 38-)
AGENT_PKGS := $(shell echo cmd/$(AGENT_NAME) && go list -f '{{ join .Deps "\n" }}' ./cmd/$(AGENT_NAME)/ | grep 'github.com/argoproj/argocd-autopilot/' | cut -c 38-)

GIT_COMMIT=$(shell git rev-parse HEAD)

ifndef GOBIN
$(error GOBIN is not set, please make sure you set your GOBIN correctly!)
endif

define docker_build
	docker build \
		--build-arg BIN_NAME=$(1) \
		--build-arg OUT_DIR=$(OUT_DIR) \
		--target $(1) \
		-t $(IMAGE_NAMESPACE)/$(1):$(VERSION) .
endef

.PHONY: all
all: binaries images

.PHONY: binaries
binaries: clis agent

.PHONY: images
images: cli-image agent-image

.PHONY: clis
clis: $(OUT_DIR)/$(CLI_NAME)-linux-amd64.gz $(OUT_DIR)/$(CLI_NAME)-linux-arm64.gz $(OUT_DIR)/$(CLI_NAME)-linux-ppc64le.gz $(OUT_DIR)/$(CLI_NAME)-linux-s390x.gz $(OUT_DIR)/$(CLI_NAME)-darwin-amd64.gz $(OUT_DIR)/$(CLI_NAME)-windows-amd64.gz

$(OUT_DIR)/$(CLI_NAME)-linux-amd64: GO_FLAGS='GOOS=linux GOARCH=amd64'
$(OUT_DIR)/$(CLI_NAME)-darwin-amd64: GO_FLAGS='GOOS=darwin GOARCH=amd64'
$(OUT_DIR)/$(CLI_NAME)-windows-amd64: GO_FLAGS='GOOS=windows GOARCH=amd64'
$(OUT_DIR)/$(CLI_NAME)-linux-arm64: GO_FLAGS='GOOS=linux GOARCH=arm64'
$(OUT_DIR)/$(CLI_NAME)-linux-ppc64le: GO_FLAGS='GOOS=linux GOARCH=ppc64le'
$(OUT_DIR)/$(CLI_NAME)-linux-s390x: GO_FLAGS='GOOS=linux GOARCH=s390x'

$(OUT_DIR)/$(CLI_NAME)-%.gz: $(OUT_DIR)/$(CLI_NAME)-%
	gzip --force --keep $(OUT_DIR)/$(CLI_NAME)-$*

$(OUT_DIR)/$(CLI_NAME)-%: $(CLI_PKGS)
	@ GO_FLAGS=$(GO_FLAGS) \
	BINARY_NAME=$(CLI_NAME) \
	VERSION=$(VERSION) \
	GIT_COMMIT=$(GIT_COMMIT) \
	OUT_FILE=$@ \
	MAIN=./cmd/$(CLI_NAME) \
	./hack/build.sh

.PHONY: cli-image
cli-image: $(OUT_DIR)/$(CLI_NAME).image

$(OUT_DIR)/$(CLI_NAME).image: $(CLI_PKGS)
	$(call docker_build,$(CLI_NAME))
	touch $(OUT_DIR)/$(CLI_NAME).image

.PHONY: agent
agent: gen-protos $(AGENT_PKGS)
	@ BINARY_NAME=$(AGENT_NAME) \
	VERSION=$(VERSION) \
	GIT_COMMIT=$(GIT_COMMIT) \
	OUT_FILE=$(OUT_DIR)/$(AGENT_NAME) \
	MAIN=./cmd/$(AGENT_NAME) \
	./hack/build.sh

.PHONY: agent-image
agent-image: $(OUT_DIR)/$(AGENT_NAME).image

$(OUT_DIR)/$(AGENT_NAME).image: $(AGENT_PKGS)
	$(call docker_build,$(AGENT_NAME))
	touch $(OUT_DIR)/$(AGENT_NAME).image

.PHONY: lint
lint: $(GOBIN)/golangci-lint
	@go mod tidy
	@echo linting go code...
	@golangci-lint run --fix --timeout 3m

.PHONY: test
test:
	./hack/test.sh

.PHONY: codegen
codegen: $(GOBIN)/mockery
	go generate ./...

.PHONY: gen-protos
gen-protos: $(GOBIN)/buf
	BIN_NAME=$(AGENT_NAME) \
	VERSION=$(VERSION) \
	./hack/proto_generate.sh

.PHONY: pre-commit
pre-commit: all lint codegen test

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

$(GOBIN)/buf: $(GOBIN)/protoc-gen-grpc-gateway $(GOBIN)/protoc-gen-openapiv2 $(GOBIN)/protoc-gen-gogofast $(GOBIN)/protoc-gen-go-grpc
	$(eval BUF_TMP := $(shell mktemp -d))
	cd $(BUF_TMP); GO111MODULE=on go get github.com/bufbuild/buf/cmd/buf@v0.39.1
	@rm -rf $(BUF_TMP)

$(GOBIN)/protoc-gen-grpc-gateway:
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway

$(GOBIN)/protoc-gen-openapiv2:
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2

$(GOBIN)/protoc-gen-gogofast:
	go install github.com/gogo/protobuf/protoc-gen-gogofast

$(GOBIN)/protoc-gen-go-grpc:
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc
