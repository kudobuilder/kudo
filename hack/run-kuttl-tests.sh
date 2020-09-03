#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

INTEGRATION_OUTPUT_JUNIT=${INTEGRATION_OUTPUT_JUNIT:-false}
TEST_ONLY=${TEST_ONLY:+"--test $TEST_ONLY"}

if [ "$INTEGRATION_OUTPUT_JUNIT" == true ]
then
    echo "Running integration tests with junit output"
    mkdir -p reports/
    go get github.com/jstemmer/go-junit-report
    go run ./cmd/kubectl-kudo test --config test/kudo-integration-test.yaml ${TEST_ONLY} 2>&1 |tee /dev/fd/2 |go-junit-report -set-exit-code > reports/kudo_test_report.xml
else
    echo "Running integration tests without junit output"
    go run ./cmd/kubectl-kudo test --config test/kudo-integration-test.yaml ${TEST_ONLY}
fi
