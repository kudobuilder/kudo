#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

# "TARGET" is a Makefile target that runs tests
TARGET=$1

INTEGRATION_OUTPUT_JUNIT=${INTEGRATION_OUTPUT_JUNIT:-false}

CONTAINER_SUFFIX=$(< /dev/urandom base64 | tr -dc '[:alpha:]' | head -c 8 || true)
CONTAINER_NAME=${CONTAINER_NAME:-"kudo-e2e-test-$CONTAINER_SUFFIX"}

# Set test harness artifacts dir to '/tmp/kudo-e2e-test', as it's easier to copy out from a container.
echo 'artifactsDir: /tmp/kudo-e2e-test' >> kudo-e2e-test.yaml

# Pull the builder image with retries if it doesn't already exist.
retries=0
builder_image=$(awk '/FROM/ {print $2}' test/Dockerfile)

function cleanup() {
    # Archive test harness artifacts
    if [ "$TARGET" == "e2e-test" ]; then
        docker cp "$CONTAINER_NAME:/tmp/kudo-e2e-test" - | bzip2 > kind-logs.tar.bz2
    fi

    docker rm "$CONTAINER_NAME"
}

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
    if docker run -e INTEGRATION_OUTPUT_JUNIT --net=host -it -m 4g \
        --name "$CONTAINER_NAME" \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -v "$(pwd)"/reports:/go/src/github.com/kudobuilder/kudo/reports \
        -v "$(pwd)"/kind-logs:/go/src/github.com/kudobuilder/kudo/kind-logs \
        kudo-test bash -c "make $TARGET ; chmod a+r -R kind-logs"
    then
        echo outside container:
        find kind-logs -ls
        id
        tar -zcvf kind-logs-exposed.tgz kind-logs
        cleanup
        echo "Tests finished successfully! ヽ(•‿•)ノ"
    else
        RESULT=$?
        echo outside container:
        find kind-logs -ls
        id
        tar -zcvf kind-logs-exposed.tgz kind-logs
        cleanup
        exit $RESULT
    fi
else
    echo "Error when building test docker image, cannot run tests."
    exit 1
fi
