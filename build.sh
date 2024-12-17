#!/bin/bash
cd "$(dirname "${BASH_SOURCE[0]}")"

set -ex

export CGO_ENABLED=0

wire ./...

rm -rf bin/
go build -o bin/ ./cmd/...