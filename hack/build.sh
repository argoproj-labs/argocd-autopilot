#!/bin/sh

if [[ ! -z "${GO_FLAGS}" ]]; then
    echo Building \"${OUT_FILE}\" with flags: \"${GO_FLAGS}\" starting at: \"${MAIN}\"
    export ${GO_FLAGS}
fi

go build -ldflags=" \
    -X 'github.com/argoproj/argocd-autopilot/pkg/store.binaryName=${BINARY_NAME}' \
    -X 'github.com/argoproj/argocd-autopilot/pkg/store.version=${VERSION}' \
    -X 'github.com/argoproj/argocd-autopilot/pkg/store.buildDate=${BUILD_DATE}' \
    -X 'github.com/argoproj/argocd-autopilot/pkg/store.gitCommit=${GIT_COMMIT}' \
    -X 'github.com/argoproj/argocd-autopilot/pkg/store.installationManifestsURL=${INSTALLATION_MANIFESTS_URL}' \
    -X 'github.com/argoproj/argocd-autopilot/pkg/store.InstallationManifestsNamespacedURL=${INSTALLATION_MANIFESTS_NAMESPACED_URL}'" \
    -v -o ${OUT_FILE} ${MAIN}