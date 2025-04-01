VERSION=v0.4.19
OUT_DIR=dist

CLI_NAME?=argocd-autopilot
IMAGE_REPOSITORY?=quay.io
IMAGE_NAMESPACE?=argoprojlabs

INSTALLATION_MANIFESTS_URL="github.com/argoproj-labs/argocd-autopilot/manifests/base?ref=$(VERSION)"
INSTALLATION_MANIFESTS_INSECURE_URL="github.com/argoproj-labs/argocd-autopilot/manifests/insecure?ref=$(VERSION)"

DEV_INSTALLATION_MANIFESTS_URL="manifests/base"
DEV_INSTALLATION_MANIFESTS_INSECURE_URL="manifests/insecure"

CLI_SRCS := $(shell find . -name '*.go')

MKDOCS_DOCKER_IMAGE?=squidfunk/mkdocs-material:4.1.1

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
	docker buildx build -t $(IMAGE_REPOSITORY)/$(IMAGE_NAMESPACE)/$(1):dev-$(VERSION) .
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
cli: $(OUT_DIR)/$(CLI_NAME)-linux-amd64.sha256 $(OUT_DIR)/$(CLI_NAME)-linux-arm64.sha256 $(OUT_DIR)/$(CLI_NAME)-linux-ppc64le.sha256 $(OUT_DIR)/$(CLI_NAME)-linux-s390x.sha256 $(OUT_DIR)/$(CLI_NAME)-darwin-amd64.sha256 $(OUT_DIR)/$(CLI_NAME)-darwin-arm64.sha256 $(OUT_DIR)/$(CLI_NAME)-windows-amd64.sha256

.PHONY: cli-local
cli-local: $(OUT_DIR)/$(CLI_NAME)-$(shell go env GOOS)-$(shell go env GOARCH)
	@cp $(OUT_DIR)/$(CLI_NAME)-$(shell go env GOOS)-$(shell go env GOARCH) /usr/local/bin/$(CLI_NAME)-dev

.PHONY: cli-package
cli-package: $(OUT_DIR)/$(CLI_NAME)-$(shell go env GOOS)-$(shell go env GOARCH)
	@cp $(OUT_DIR)/$(CLI_NAME)-$(shell go env GOOS)-$(shell go env GOARCH) $(OUT_DIR)/$(CLI_NAME)

$(OUT_DIR)/$(CLI_NAME)-linux-amd64: GO_FLAGS='GOOS=linux GOARCH=amd64 CGO_ENABLED=0'
$(OUT_DIR)/$(CLI_NAME)-darwin-amd64: GO_FLAGS='GOOS=darwin GOARCH=amd64 CGO_ENABLED=0'
$(OUT_DIR)/$(CLI_NAME)-darwin-arm64: GO_FLAGS='GOOS=darwin GOARCH=arm64 CGO_ENABLED=0'
$(OUT_DIR)/$(CLI_NAME)-windows-amd64.exe: GO_FLAGS='GOOS=windows GOARCH=amd64 CGO_ENABLED=0'
$(OUT_DIR)/$(CLI_NAME)-linux-arm64: GO_FLAGS='GOOS=linux GOARCH=arm64 CGO_ENABLED=0'
$(OUT_DIR)/$(CLI_NAME)-linux-ppc64le: GO_FLAGS='GOOS=linux GOARCH=ppc64le CGO_ENABLED=0'
$(OUT_DIR)/$(CLI_NAME)-linux-s390x: GO_FLAGS='GOOS=linux GOARCH=s390x CGO_ENABLED=0'

# special target for windows, to add the ".exe" extension at the end
$(OUT_DIR)/$(CLI_NAME)-windows-amd64.tar.gz:
	@make $(OUT_DIR)/$(CLI_NAME)-windows-amd64.exe
	cd $(OUT_DIR) && tar -czvf $(CLI_NAME)-windows-amd64.tar.gz $(CLI_NAME)-windows-amd64.exe && cd ..

$(OUT_DIR)/$(CLI_NAME)-%.tar.gz:
	@make $(OUT_DIR)/$(CLI_NAME)-$*
	cd $(OUT_DIR) && tar -czvf $(CLI_NAME)-$*.tar.gz $(CLI_NAME)-$* && cd ..

$(OUT_DIR)/$(CLI_NAME)-%.sha256:
	@make $(OUT_DIR)/$(CLI_NAME)-$*.tar.gz
	openssl dgst -sha256 "$(OUT_DIR)/$(CLI_NAME)-$*.tar.gz" | awk '{ print $$2 }' > "$(OUT_DIR)/$(CLI_NAME)-$*".sha256

$(OUT_DIR)/$(CLI_NAME)-%: $(CLI_SRCS)
	@GO_FLAGS=$(GO_FLAGS) \
	BUILD_DATE=$(BUILD_DATE) \
	BINARY_NAME=$(CLI_NAME) \
	VERSION=$(VERSION) \
	GIT_COMMIT=$(GIT_COMMIT) \
	OUT_FILE=$(OUT_DIR)/$(CLI_NAME)-$* \
	INSTALLATION_MANIFESTS_URL=$(INSTALLATION_MANIFESTS_URL) \
	INSTALLATION_MANIFESTS_NAMESPACED_URL=$(INSTALLATION_MANIFESTS_NAMESPACED_URL) \
	MAIN=./cmd \
	./hack/build.sh

.PHONY: cli-image
cli-image: tidy $(OUT_DIR)/$(CLI_NAME).image

$(OUT_DIR)/$(CLI_NAME).image: $(CLI_SRCS)
	$(call docker_build,$(CLI_NAME))
	@mkdir -p $(OUT_DIR)
	@touch $(OUT_DIR)/$(CLI_NAME).image

.PHONY: lint
lint: tidy golangci-lint
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: tidy golangci-lint
	$(GOLANGCI_LINT) run --fix

.PHONY: test
test:
	./hack/test.sh

.PHONY: codegen
codegen: mockgen
	rm -f docs/commands/*
	@PATH=$(LOCALBIN):$(PATH) go generate ./...

.PHONY: pre-commit
pre-commit: lint

.PHONY: pre-push
pre-push: tidy lint test codegen check-worktree

.PHONY: build-docs
build-docs:
	docker run ${MKDOCS_RUN_ARGS} --rm -it -p 8000:8000 -v $(shell pwd):/docs ${MKDOCS_DOCKER_IMAGE} build

.PHONY: serve-docs
serve-docs:
	docker run ${MKDOCS_RUN_ARGS} --rm -it -p 8000:8000 -v $(shell pwd):/docs ${MKDOCS_DOCKER_IMAGE} serve -a 0.0.0.0:8000

.PHONY: release
release: tidy check-worktree fetch-tags
	./hack/release.sh

.PHONY: fetch-tags
fetch-tags:
	git fetch --tags

.PHONY: clean
clean:
	@rm -rf dist

.PHONY: tidy
tidy:
	@echo running go mod tidy...
	@go mod tidy

.PHONY: check-worktree
check-worktree:
	@./hack/check_worktree.sh

## copied binary dependencies handling from kubebuilder generated code
## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
MOCKGEN = ${LOCALBIN}/mockgen

## Tool Versions
MOCKGEN_VERSION ?= v1.6.0
GOLANGCI_LINT_VERSION ?= v1.64.7

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT):
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

.PHONY: mockgen
mockgen: $(MOCKGEN) ## Download mockgen locally if necessary.
$(MOCKGEN): $(LOCALBIN)
	$(call go-install-tool,$(MOCKGEN),github.com/golang/mock/mockgen,$(MOCKGEN_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef
