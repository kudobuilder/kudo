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
    (
        cd "${CODE_GEN_DIR}"
        # Make sure code-gen uses the local kudo codebase, and not a remote module
        go mod edit -replace=github.com/kudobuilder/kudo=../../../
    )
fi

# Fake being in a $GOPATH until kubernetes fully supports modules
# https://github.com/kudobuilder/kudo/issues/1252
FAKE_GOPATH="$(mktemp -d)"
trap 'chmod -R u+rwX ${FAKE_GOPATH} && rm -rf ${FAKE_GOPATH}' EXIT
FAKE_REPOPATH="${FAKE_GOPATH}/src/github.com/kudobuilder/kudo"
mkdir -p "$(dirname "${FAKE_REPOPATH}")" && ln -s "${REPO_ROOT}" "${FAKE_REPOPATH}"
export GOPATH="${FAKE_GOPATH}"
cd "${FAKE_REPOPATH}"

#"${CODE_GEN_DIR}"/generate-groups.sh \
#  all \
#  github.com/kudobuilder/kudo/pkg/client \
#  github.com/kudobuilder/kudo/pkg/apis \
#  "kudo:v1beta1,v1beta2" \
#  --go-header-file hack/boilerplate.go.txt # must be last for some reason

# Generation is split into two parts, in case we want to generate clients for different APIs than for internal conversion

# This part is for generating the client, lister, informer, i.e. everything client related
#"${CODE_GEN_DIR}"/generate-groups.sh \
#  deepcopy \
#  github.com/kudobuilder/kudo/pkg/client \
#  github.com/kudobuilder/kudo/pkg/apis \
#  "kudo:v1beta2" \
#  --go-header-file ${REPO_ROOT}/hack/boilerplate.go.txt # must be last for some reason

# Execute this in a subshell - generate-internal-groups expects the cwd to be the code_gen_dir
(
    # This part is for generating the internal conversion, defaulting, etc.
    cd "${CODE_GEN_DIR}"
    ./generate-internal-groups.sh \
      deepcopy,defaulter,conversion \
      github.com/kudobuilder/kudo/pkg/client \
      github.com/kudobuilder/kudo/pkg/apis \
      github.com/kudobuilder/kudo/pkg/apis \
      "kudo:v1beta1,v1beta2" \
      --go-header-file ${REPO_ROOT}/hack/boilerplate.go.txt # must be last for some reason
)
