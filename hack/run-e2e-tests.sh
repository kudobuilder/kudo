#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

INTEGRATION_OUTPUT_JUNIT=${INTEGRATION_OUTPUT_JUNIT:-false}
KUDO_VERSION=${KUDO_VERSION:-test}

docker build . \
    --build-arg ldflags_arg="" \
    -t "kudobuilder/controller:$KUDO_VERSION"

sed "s/%version%/$KUDO_VERSION/" kudo-e2e-test.yaml.tmpl > kudo-e2e-test.yaml
sed "s/%version%/$KUDO_VERSION/" kudo-upgrade-test.yaml.tmpl > kudo-upgrade-test.yaml

if [ "$INTEGRATION_OUTPUT_JUNIT" == true ]
then
    echo "Running E2E tests with junit output"
    mkdir -p reports/
    go get github.com/jstemmer/go-junit-report

    ./bin/kubectl-kudo test --config kudo-e2e-test.yaml 2>&1 \
        | tee /dev/fd/2 \
        | go-junit-report -set-exit-code \
        > reports/kudo_e2e_test_report.xml

    ./bin/kubectl-kudo test --config kudo-upgrade-test.yaml 2>&1 \
        | tee /dev/fd/2 \
        | go-junit-report -set-exit-code \
        > reports/kudo_upgrade_test_report.xml

    # Operators tests
    rm -rf operators
    git clone https://github.com/kudobuilder/operators
    mkdir operators/bin/
    cp ./bin/kubectl-kudo operators/bin/
    sed "s/%version%/$KUDO_VERSION/" operators/kudo-test.yaml.tmpl > operators/kudo-test.yaml
    cd operators && ./bin/kubectl-kudo test --artifacts-dir /tmp/kudo-e2e-test 2>&1 \
        | tee /dev/fd/2 \
        | go-junit-report -set-exit-code \
        > ../reports/kudo_operators_test_report.xml
else
    echo "Running E2E tests without junit output"

    ./bin/kubectl-kudo test --config kudo-e2e-test.yaml

    ./bin/kubectl-kudo test --config kudo-upgrade-test.yaml

    # Operators tests
    rm -rf operators
    git clone https://github.com/kudobuilder/operators
    mkdir operators/bin/
    cp ./bin/kubectl-kudo operators/bin/
    sed "s/%version%/$KUDO_VERSION/" operators/kudo-test.yaml.tmpl > operators/kudo-test.yaml
    cd operators && ./bin/kubectl-kudo test --artifacts-dir /tmp/kudo-e2e-test
fi
