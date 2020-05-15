#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

INTEGRATION_OUTPUT_JUNIT=${INTEGRATION_OUTPUT_JUNIT:-false}
VERSION=${VERSION:-test}

docker build . \
    --build-arg ldflags_arg="" \
    -t "kudobuilder/controller:$VERSION"

# Generate the kudo.yaml that is used to install KUDO while running e2e-test
./bin/kubectl-kudo init --webhook InstanceValidation \
    --unsafe-self-signed-webhook-ca --dry-run --output yaml \
    --kudo-image kudobuilder/controller:$VERSION \
    --kudo-image-pull-policy Never \
    > test/manifests/kudo.yaml

sed "s/%version%/$VERSION/" kudo-e2e-test.yaml.tmpl > kudo-e2e-test.yaml

if [ "$INTEGRATION_OUTPUT_JUNIT" == true ]
then
    echo "Running E2E tests with junit output"
    mkdir -p reports/
    go get github.com/jstemmer/go-junit-report

    ./bin/kubectl-kudo test --config kudo-e2e-test.yaml 2>&1 \
        | tee /dev/fd/2 \
        | go-junit-report -set-exit-code \
        > reports/kudo_e2e_test_report.xml

    rm -rf operators
    git clone https://github.com/kudobuilder/operators
    mkdir operators/bin/
    cp ./bin/kubectl-kudo operators/bin/
    cp ./bin/manager operators/bin/
    cd operators && ./bin/kubectl-kudo test --artifacts-dir /tmp/kudo-e2e-test 2>&1 \
        | tee /dev/fd/2 \
        | go-junit-report -set-exit-code \
        > ../reports/kudo_operators_test_report.xml
else
    echo "Running E2E tests without junit output"

    ./bin/kubectl-kudo test --config kudo-e2e-test.yaml

    rm -rf operators
    git clone https://github.com/kudobuilder/operators
    mkdir operators/bin/
    cp ./bin/kubectl-kudo operators/bin/
    cp ./bin/manager operators/bin/
    cd operators && ./bin/kubectl-kudo test --artifacts-dir /tmp/kudo-e2e-test
fi
