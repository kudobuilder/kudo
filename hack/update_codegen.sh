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
set GO111MODULE=on
VERSION=$(go list -m all | grep k8s.io/code-generator | rev | cut -d"-" -f1 | cut -d" " -f1 | rev)
git clone https://github.com/kubernetes/code-generator.git ${TMP_DIR}
(cd ${TMP_DIR} && git reset --hard ${VERSION} && go mod init)
${TMP_DIR}/generate-groups.sh \
  all \
  github.com/kudobuilder/kudo/pkg/client \
  github.com/kudobuilder/kudo/pkg/apis \
  "kudo:v1alpha1" \
  --go-header-file hack/boilerplate.go.txt # must be last for some reason