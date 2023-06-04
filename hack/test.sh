#!/bin/sh

set -e
echo "" > coverage.txt

go mod tidy
git status

echo "before github.com/argoproj-labs/argocd-autopilot/cmd/commands"
go test -v -race -coverprofile=profile.out -covermode=atomic github.com/argoproj-labs/argocd-autopilot/cmd/commands
echo "after github.com/argoproj-labs/argocd-autopilot/cmd/commands"

# for d in $(go list ./... | grep -v vendor); do
#     go test -v -race -coverprofile=profile.out -covermode=atomic $d
#     if [ -f profile.out ]; then
#         cat profile.out >> coverage.txt
#         rm profile.out
#     fi
# done

