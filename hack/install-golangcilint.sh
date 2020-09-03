#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

GOLANGCILINT_VERSION=${GOLANGCILINT_VERSION:-v1.29.0}

curl -sSfL "https://raw.githubusercontent.com/golangci/golangci-lint/${GOLANGCILINT_VERSION}/install.sh" | sh -s -- -b "$(go env GOPATH)/bin" "${GOLANGCILINT_VERSION}"
