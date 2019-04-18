#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

TMP_DIR=$(mktemp -d)

cleanup() {
  rm -rf "${TMP_DIR}"
}
trap "cleanup" EXIT SIGINT

cleanup

# The following solution for making code generation work with go modules is
# borrowed and modified from https://github.com/heptio/contour/pull/1010.
VERSION=$(go list -m all | grep sigs.k8s.io/controller-tools | rev | cut -d"-" -f1 | cut -d" " -f1 | rev)
git clone https://github.com/kubernetes-sigs/controller-tools.git ${TMP_DIR}
(cd ${TMP_DIR} && git reset --hard ${VERSION})

set GO111MODULE=on
go run $TMP_DIR/cmd/controller-gen/main.go rbac --output-dir=config/default/rbac
go run $TMP_DIR/cmd/controller-gen/main.go crd
