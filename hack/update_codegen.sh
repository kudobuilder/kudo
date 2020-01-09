#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Get absolute directory of this script
HACK_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

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
    (
        cd "${CODE_GEN_DIR}"
        # Make sure code-gen uses the local kudo codebase, and not a remote module
        go mod edit -replace=github.com/kudobuilder/kudo=../../../
    )
fi

# Generation is split into two parts, in case we want to generate clients for different APIs than for internal conversion

# This part is for generating the client, lister, informer, i.e. everything client related
"${CODE_GEN_DIR}"/generate-groups.sh \
  all \
  github.com/kudobuilder/kudo/pkg/client \
  github.com/kudobuilder/kudo/pkg/apis \
  "kudo:v1beta1" \
  --go-header-file ${HACK_DIR}/boilerplate.go.txt # must be last for some reason

# Execute this in a subshell - generate-internal-groups expects the cwd to be the code_gen_dir
(
    # This part is for generating the internal conversion, defaulting, etc.
    cd "${CODE_GEN_DIR}"
    ./generate-internal-groups.sh \
      deepcopy,defaulter,conversion \
      github.com/kudobuilder/kudo/pkg/client \
      github.com/kudobuilder/kudo/pkg/apis \
      github.com/kudobuilder/kudo/pkg/apis \
      "kudo:v1beta1" \
      --go-header-file ${HACK_DIR}/boilerplate.go.txt # must be last for some reason
)
