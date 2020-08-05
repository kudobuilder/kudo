#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# The following solution for making code generation work with go modules is
# borrowed and modified from https://github.com/heptio/contour/pull/1010.
# it has been modified to enable caching.
export GO111MODULE=on
version_and_repo=$(go list -f '{{if .Replace}}{{.Replace.Version}} {{.Replace.Path}}{{else}}{{.Version}} {{.Path}}{{end}}' -m k8s.io/code-generator)
VERSION="$(echo "${version_and_repo}" | cut -d' ' -f 1 | rev | cut -d"-" -f1 | rev)"
REPO_ROOT="${REPO_ROOT:-$(git rev-parse --show-toplevel)}"
CODE_GEN_DIR="${REPO_ROOT}/hack/code-gen/$VERSION"

if [[ -d ${CODE_GEN_DIR} ]]; then
    echo "Using cached code generator version: $VERSION"
else
    git clone https://github.com/kubernetes/code-generator.git "${CODE_GEN_DIR}"
    git -C "${CODE_GEN_DIR}" reset --hard "${VERSION}"
fi

# Fake being in a $GOPATH until kubernetes fully supports modules
# https://github.com/kudobuilder/kudo/issues/1252
FAKE_GOPATH="$(mktemp -d)"
trap 'chmod -R u+rwX ${FAKE_GOPATH} && rm -rf ${FAKE_GOPATH}' EXIT
FAKE_REPOPATH="${FAKE_GOPATH}/src/github.com/kudobuilder/kudo"
mkdir -p "$(dirname "${FAKE_REPOPATH}")" && ln -s "${REPO_ROOT}" "${FAKE_REPOPATH}"
export GOPATH="${FAKE_GOPATH}"
cd "${FAKE_REPOPATH}"

"${CODE_GEN_DIR}"/generate-groups.sh \
  all \
  github.com/kudobuilder/kudo/pkg/client \
  github.com/kudobuilder/kudo/pkg/apis \
  "kudo:v1beta1" \
  --go-header-file hack/boilerplate.go.txt # must be last for some reason
