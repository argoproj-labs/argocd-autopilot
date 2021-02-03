#!/bin/sh

go build -ldflags=" \
    -X 'github.com/codefresh-io/cf-argo/pkg/store.binaryName=${BINARY_NAME}' \
    -X 'github.com/codefresh-io/cf-argo/pkg/store.version=${VERSION}' \
    -X 'github.com/codefresh-io/cf-argo/pkg/store.gitCommit=${GIT_COMMIT}' \
    -X 'github.com/codefresh-io/cf-argo/pkg/store.baseGitURL=${BASE_GIT_URL}'" \
    -v -o ${OUT_DIR}/${BINARY_NAME} .