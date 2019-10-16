#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

INTEGRATION_OUTPUT_JUNIT=${INTEGRATION_OUTPUT_JUNIT:-false}

# Pull the builder image with retries if it doesn't already exist.
retries=0
builder_image=$(awk '/FROM/ {print $2}' test/Dockerfile)

if ! docker inspect "$builder_image"; then
    until docker pull "$builder_image"; do
        if [ $retries -eq 3 ]; then
            echo "Giving up downloading builder image, failing build."
            exit 1
        fi
        echo "Docker pull failed, retrying."
        ((retries++))
        sleep 1
    done
fi

if docker build -f test/Dockerfile -t kudo-test .; then
    if docker run -e INTEGRATION_OUTPUT_JUNIT -it -m 4g -v "$(pwd)"/reports:/go/src/github.com/kudobuilder/kudo/reports --rm kudo-test; then
        echo "Tests finished successfully! ヽ(•‿•)ノ"
    else
        exit $?
    fi
else
    echo "Error when building test docker image, cannot run tests."
    exit 1
fi
