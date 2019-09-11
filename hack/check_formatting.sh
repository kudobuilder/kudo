#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(dirname "${BASH_SOURCE[0]}")/..
PACKAGES="${ROOT}/pkg/ ${ROOT}/cmd/"

# Make sure goimports doesn't find any errors
go install golang.org/x/tools/cmd/goimports

echo "goimports -d ${PACKAGES}"
# shellcheck disable=SC2086
differences=$(goimports -d ${PACKAGES})
if [[ ! "$differences" == "" ]]; then
    echo "goimports found the following differences"
    echo "$differences"
    exit 1
fi
