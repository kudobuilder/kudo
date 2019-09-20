#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# The following solution for making code generation work with go modules is
# borrowed and modified from https://github.com/heptio/contour/pull/1010.
# it has been modified to enable caching.
export GO111MODULE=on

ROOT=$(dirname "${BASH_SOURCE[0]}")/..

cd "$ROOT"

go run cmd/schema-gen/main.go config/schemas

cd -