#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

INTEGRATION_OUTPUT_JUNIT=${INTEGRATION_OUTPUT_JUNIT:-false}

function run_kudo_test () {
    local config="$1"
    local report="$2"

    if [ "$INTEGRATION_OUTPUT_JUNIT" == true ]; then
        mkdir -p reports/
        go run ./cmd/kubectl-kudo test --config "$config" 2>&1 | tee /dev/fd/2 | go-junit-report -set-exit-code > "$report"
    else
        go run ./cmd/kubectl-kudo test --config "$config"
    fi
}

if [ "$INTEGRATION_OUTPUT_JUNIT" == true ]; then
    echo "Running E2E tests with junit output"
    go get github.com/jstemmer/go-junit-report
else
     echo "Running E2E tests without junit output"
fi

    run_kudo_test kudo-e2e-test.yaml reports/kudo_e2e_test_report.xml

    rm -rf operators
    git clone https://github.com/kudobuilder/operators
    mkdir operators/bin/
    cp ./bin/kubectl-kudo operators/bin/
    cd operators && run_kudo_test kudo-test.yaml ../reports/kudo_operators_test_report.xml
