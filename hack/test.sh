#!/bin/sh

set -e
echo "" > coverage.txt

for d in $(go list ./...); do
    echo "tidying"
    # go mod tidy
    # git status
    echo "before $d"
    # go test -v -race -coverprofile=profile.out -covermode=atomic $d
    go test -v $d
    echo "after $d"
    # if [ -f profile.out ]; then
    #     cat profile.out >> coverage.txt
    #     rm profile.out
    # fi
done

