#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# The following solution for making code generation work with go modules is
# borrowed and modified from https://github.com/heptio/contour/pull/1010.
# it has been modified to enable caching.
export GO111MODULE=on
VERSION=$(go list -m all | grep k8s.io/code-generator | rev | cut -d"-" -f1 | cut -d" " -f1 | rev)
CODE_GEN_DIR="hack/code-gen/$VERSION"

if [[ -d ${CODE_GEN_DIR} ]]; then
    echo "Using cached code generator version: $VERSION"
else
    git clone https://github.com/kubernetes/code-generator.git "${CODE_GEN_DIR}"
    git -C "${CODE_GEN_DIR}" reset --hard "${VERSION}"
fi

"${CODE_GEN_DIR}"/generate-groups.sh \
  deepcopy \
  github.com/kudobuilder/kudo/pkg/client \
  github.com/kudobuilder/kudo/pkg/apis \
  "testharness:v1beta1" \
  --go-header-file hack/boilerplate.go.txt # must be last for some reason

"${CODE_GEN_DIR}"/generate-groups.sh \
  all \
  github.com/kudobuilder/kudo/pkg/client \
  github.com/kudobuilder/kudo/pkg/apis \
  "kudo:v1beta1" \
  --go-header-file hack/boilerplate.go.txt # must be last for some reason
