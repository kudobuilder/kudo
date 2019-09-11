#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# The following solution for making code generation work with go modules is
# borrowed and modified from https://github.com/heptio/contour/pull/1010.
# it has been modified to enable caching.
export GO111MODULE=on
VERSION=$(go list -m all | grep sigs.k8s.io/controller-tools | rev | cut -d"-" -f1 | cut -d" " -f1 | rev)
CONTROLLER_GEN_DIR="hack/controller-gen/$VERSION"

if [ -d "${CONTROLLER_GEN_DIR}" ]; then
    echo "Using cached controller generator version: $VERSION"
else
    git clone https://github.com/kubernetes-sigs/controller-tools.git "${CONTROLLER_GEN_DIR}"
    (cd "${CONTROLLER_GEN_DIR}" && git reset --hard "${VERSION}" && go mod init)
fi

go run "$CONTROLLER_GEN_DIR"/cmd/controller-gen/main.go rbac
go run "$CONTROLLER_GEN_DIR"/cmd/controller-gen/main.go crd --domain dev
