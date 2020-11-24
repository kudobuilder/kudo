#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

# "TARGET" is a Makefile target that runs tests
TARGET=$1

INTEGRATION_OUTPUT_JUNIT=${INTEGRATION_OUTPUT_JUNIT:-false}

function archive_logs() {
    # Archive test harness artifacts
    if [[ "$TARGET" == "e2e-test" ]] || [[ "$TARGET" == "operator-test" ]] || [[ "$TARGET" == "upgrade-test" ]]; then
        tar -cjvf kind-logs.tar.bz2 reports/kind-logs/
    fi
}

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
    if docker run -e INTEGRATION_OUTPUT_JUNIT -e TEST --net=host -it --rm -m 4g \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -v "$(pwd)"/reports:/go/src/github.com/kudobuilder/kudo/reports \
        kudo-test bash -c "make $TARGET; ret=\$?; chmod a+r -R /go/src/github.com/kudobuilder/kudo/reports; exit \$ret"
    then
        archive_logs
        echo "Tests finished successfully! ヽ(•‿•)ノ"
    else
        RESULT=$?
        archive_logs
        exit $RESULT
    fi
else
    echo "Error when building test docker image, cannot run tests."
    exit 1
fi
