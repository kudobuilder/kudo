#!/bin/bash

set -x -e -o pipefail

ROOT=$(dirname "${BASH_SOURCE}")/..
PACKAGES="${ROOT}/pkg/ ${ROOT}/cmd/"

# Make sure goimports doesn't find any errors
echo "goimports -d ${PACKAGES}"
differences=`goimports -d ${PACKAGES}`
if [[ ! "$differences" == "" ]]; then
    echo "goimports found the following differences"
    echo "$differences"
    exit 1
fi