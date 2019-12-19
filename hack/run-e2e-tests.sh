#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

INTEGRATION_OUTPUT_JUNIT=${INTEGRATION_OUTPUT_JUNIT:-false}

if [ "$INTEGRATION_OUTPUT_JUNIT" == true ]
then
    echo "Running E2E tests with junit output"
    mkdir -p reports/
    go get github.com/jstemmer/go-junit-report

    go run ./cmd/kubectl-kudo test --config kudo-e2e-test.yaml 2>&1 |tee /dev/fd/2 |go-junit-report -set-exit-code > reports/kudo_e2e_test_report.xml

	rm -rf operators
	git clone https://github.com/kudobuilder/operators
	mkdir operators/bin/
	cp ./bin/kubectl-kudo operators/bin/
	cd operators && go run ../cmd/kubectl-kudo test 2>&1 |tee /dev/fd/2 |go-junit-report -set-exit-code > ../reports/kudo_operators_test_report.xml
else
    echo "Running E2E tests without junit output"

    go run ./cmd/kubectl-kudo test --config kudo-e2e-test.yaml

	rm -rf operators
	git clone https://github.com/kudobuilder/operators
	mkdir operators/bin/
	cp ./bin/kubectl-kudo operators/bin/
	cd operators && go run ../cmd/kubectl-kudo test
fi
