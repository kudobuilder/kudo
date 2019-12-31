#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# make sure make generate can be invoked
if ! make generate; then
    echo "Invoking 'make generate' ends with non-zero exit code."
    exit 1
fi

if ! git diff-index --quiet HEAD --; then
    echo "Running 'make generate' produces changes to the current git status. Maybe you forgot to check-in your updated generated files?"
    echo "The current diff: `git diff`"
    exit 1
fi

echo "Verifying 'make generate' was successful! ヽ(•‿•)ノ"
exit 0