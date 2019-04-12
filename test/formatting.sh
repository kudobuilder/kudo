#!/bin/bash

ROOT=$(dirname "${BASH_SOURCE}")/..
PACKAGES="${ROOT}/pkg/ ${ROOT}/cmd/"

# Make sure go fmt doesn't change anything
echo "gofmt -d ${PACKAGES}"
differences=`gofmt -d ${PACKAGES}`
if [[ ! "$differences" == "" ]]; then
    echo "gofmt found the following differences"
    echo "$differences"
    exit 1
fi

# Make sure goimports doesn't find any errors
echo "goimports -d ${PACKAGES}"
differences=`goimports -d ${PACKAGES}`
if [[ ! "$differences" == "" ]]; then
    echo "goimports found the following differences"
    echo "$differences"
    exit 1
fi