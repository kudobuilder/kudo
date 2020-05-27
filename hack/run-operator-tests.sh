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

rm -rf operators
git clone https://github.com/kudobuilder/operators
mkdir operators/bin/
cp ./bin/kubectl-kudo operators/bin/
sed "s/%version%/$KUDO_VERSION/" operators/kudo-test.yaml.tmpl > operators/kudo-test.yaml

if [ "$INTEGRATION_OUTPUT_JUNIT" == true ]
then
    echo "Running operator tests with junit output"
    mkdir -p reports/
    go get github.com/jstemmer/go-junit-report

    cd operators && ./bin/kubectl-kudo test --artifacts-dir /tmp/kudo-e2e-test 2>&1 \
        | tee /dev/fd/2 \
        | go-junit-report -set-exit-code \
        > ../reports/kudo_operators_test_report.xml
else
    echo "Running operator tests without junit output"

    cd operators && ./bin/kubectl-kudo test --artifacts-dir /tmp/kudo-e2e-test
fi
