#!/bin/sh

set -e
echo "" > coverage.txt

echo "before github.com/argoproj-labs/argocd-autopilot/hack/cmd-docs"
go test -v -race -coverprofile=profile.out -covermode=atomic github.com/argoproj-labs/argocd-autopilot/hack/cmd-docs
echo "after github.com/argoproj-labs/argocd-autopilot/hack/cmd-docs"

# for d in $(go list ./... | grep -v vendor); do
#     go test -v -race -coverprofile=profile.out -covermode=atomic $d
#     if [ -f profile.out ]; then
#         cat profile.out >> coverage.txt
#         rm profile.out
#     fi
# done

