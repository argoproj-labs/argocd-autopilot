#!/bin/sh

set -e
echo "" > coverage.txt

go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
