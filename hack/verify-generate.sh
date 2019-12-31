#!/usr/bin/env bash

set -o nounset
set -o pipefail
# intentionally not setting 'set -o errexit' because we want to print custom error messages

# make sure make generate can be invoked
make generate
RETVAL=$?
if [[ ${RETVAL} != 0 ]]; then
    echo "Invoking 'make generate' ends with non-zero exit code."
    exit 1
fi

git diff --exit-code --quiet
RETVAL=$?

if [[ ${RETVAL} != 0 ]]; then
    echo "Running 'make generate' produces changes to the current git status. Maybe you forgot to check-in your updated generated files?"
    echo "The current diff: `git diff`"
    exit 1
fi

echo "Verifying 'make generate' was successful! ヽ(•‿•)ノ"
exit 0