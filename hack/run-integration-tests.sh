#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

INTEGRATION_OUTPUT_JUNIT=${INTEGRATION_OUTPUT_JUNIT:-false}

MOD_FLAGS="-mod=readonly"

# When run from a Goland/IntelliJ terminal, Goland/IntelliJ already set '-mod=readonly'
if [ "${_INTELLIJ_FORCE_SET_GOFLAGS+x}" ]
then
   MOD_FLAGS=""
fi

if [ "$INTEGRATION_OUTPUT_JUNIT" == true ]
then
    echo "Running go integration tests with junit output"
    mkdir -p reports/
    go get github.com/jstemmer/go-junit-report
    go test -tags integration ./pkg/... ./cmd/... -v ${MOD_FLAGS} -coverprofile cover-integration.out 2>&1 |tee /dev/fd/2 |go-junit-report -set-exit-code > reports/integration_report.xml
else
    echo "Running integration tests without junit output"
    go test -tags integration ./pkg/... ./cmd/... -v ${MOD_FLAGS} -coverprofile cover-integration.out
fi

echo "Running KUTTL integration tests"

./hack/run-kuttl-tests.sh