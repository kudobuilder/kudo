#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

INTEGRATION_OUTPUT_JUNIT=${INTEGRATION_OUTPUT_JUNIT:-false}
KUDO_VERSION=${KUDO_VERSION:-test}
TEST_ONLY=${TEST_ONLY:+"--test $TEST_ONLY"}

docker build . \
    --build-arg ldflags_arg="" \
    -t "kudobuilder/controller:$KUDO_VERSION"

sed "s/%version%/$KUDO_VERSION/" test/kudo-upgrade-test.yaml.tmpl > test/kudo-upgrade-test.yaml

if [ "$INTEGRATION_OUTPUT_JUNIT" == true ]
then
    echo "Running Upgrade tests with junit output"
    mkdir -p reports/
    go get github.com/jstemmer/go-junit-report

    ./bin/kubectl-kudo test --config test/kudo-upgrade-test.yaml ${TEST_ONLY} 2>&1 \
        | tee /dev/fd/2 \
        | go-junit-report -set-exit-code \
        > reports/kudo_upgrade_test_report.xml
else
    echo "Running Upgrade tests without junit output"

    ./bin/kubectl-kudo test --config test/kudo-upgrade-test.yaml ${TEST_ONLY}
fi
