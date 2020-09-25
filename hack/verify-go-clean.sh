#!/usr/bin/env bash

#####
#  Used to ensure that running `go mod tidy` results in a non-dirty repository.
#  Used to verify that dependency changes have been cleaned
#####

set -o nounset
set -o pipefail
# intentionally not setting 'set -o errexit' because we want to print custom error messages

# versions of tools
echo "$(go version)"

# make sure make generate can be invoked
go mod tidy
RETVAL=$?
if [[ ${RETVAL} != 0 ]]; then
    echo "Invoking 'go mod tidy' ends with non-zero exit code."
    exit 1
fi

git diff --exit-code --quiet
RETVAL=$?

if [[ ${RETVAL} != 0 ]]; then
    echo "Running 'go mod tidy' produces changes to the current git status. Maybe you forgot to clean go.mod after changing dependencies?"
    echo "The current diff:"
    git diff
    exit 1
fi

echo "Verifying 'go mod tidy' was successful! ヽ(•‿•)ノ"
exit 0
