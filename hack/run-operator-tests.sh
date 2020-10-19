#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

# Operator tests have the same setup but different concerns from E2E tests. 
# The motivation is to get earlier feedback when these tests fail. In the past, the operator tests were only run if the E2E tests succeeded. 
# This made it hard for larger PRs to distinguish between changes that fix operators and changes that fix E2E tests.

INTEGRATION_OUTPUT_JUNIT=${INTEGRATION_OUTPUT_JUNIT:-false}
# Exposing `KUDO_VERSION` allows for KUDO_VERSION to be overriden. The default value of test is used as a tag for the Docker image that is build and injected to the kind cluster. 
# This is to avoid clashes with existing Docker images as test is only used in this context.
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

    cd operators && ./bin/kubectl-kudo test --artifacts-dir ../reports/kind-logs 2>&1 \
        | tee /dev/fd/2 \
        | go-junit-report -set-exit-code \
        > ../reports/kudo_operators_test_report.xml
else
    echo "Running operator tests without junit output"

    cd operators && ./bin/kubectl-kudo test --artifacts-dir ../reports/kind-logs
fi
