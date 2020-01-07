#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

# "TARGET" is a Makefile target that runs tests
TARGET=$1

INTEGRATION_OUTPUT_JUNIT=${INTEGRATION_OUTPUT_JUNIT:-false}

# Set test harness artifacts dir to '/tmp/kudo-e2e-test', as it's easier to copy from in a container.
echo 'artifactsDir: /tmp/kudo-e2e-test' >> kudo-e2e-test.yaml

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
    docker run -e INTEGRATION_OUTPUT_JUNIT --net=host -it -m 4g \
        --name kudo-e2e-test \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -v "$(pwd)"/reports:/go/src/github.com/kudobuilder/kudo/reports \
        kudo-test make "$TARGET"
    RESULT=$?

    # Archive test harness artifacts
    if [ "$TARGET" == "e2e-test" ]; then
        docker cp kudo-e2e-test:/tmp/kudo-e2e-test - | bzip2 > kind-logs.tar.bz2
    fi

    docker rm kudo-e2e-test

    if [ $RESULT -eq 0 ]; then
        echo "Tests finished successfully! ヽ(•‿•)ノ"
    else
        exit $RESULT
    fi
else
    echo "Error when building test docker image, cannot run tests."
    exit 1
fi
