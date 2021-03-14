#!/bin/sh

if [[ ! -z "${GO_FLAGS}" ]]; then
    echo Building with flags: ${GO_FLAGS}
    export ${GO_FLAGS}
fi

go build -ldflags=" \
    -X 'github.com/argoproj/argocd-autopilot/pkg/store.binaryName=${BINARY_NAME}' \
    -X 'github.com/argoproj/argocd-autopilot/pkg/store.version=${VERSION}' \
    -X 'github.com/argoproj/argocd-autopilot/pkg/store.gitCommit=${GIT_COMMIT}' \
    -X 'github.com/argoproj/argocd-autopilot/pkg/store.baseGitURL=${BASE_GIT_URL}'" \
    -v -i -o ${OUT_FILE} ${MAIN}