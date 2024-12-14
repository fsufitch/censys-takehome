#!/bin/bash
cd "$(dirname "${BASH_SOURCE[0]}")"

set -ex

go run github.com/google/wire/cmd/wire@latest ./...

rm -rf bin/
go build -v -o bin/ ./cmd/...