#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

INTEGRATION_OUTPUT_JUNIT=${INTEGRATION_OUTPUT_JUNIT:-false}
VERSION=${VERSION:-test}
PREV_KUDO_VERSION=0.11.1

docker build . \
    --build-arg ldflags_arg="" \
    -t "kudobuilder/controller:$VERSION"

# Generate the kudo.yaml that is used to install KUDO while running e2e-test
./bin/kubectl-kudo init --dry-run --output yaml --kudo-image kudobuilder/controller:$VERSION --kudo-image-pull-policy Never \
    > test/manifests/kudo.yaml

sed "s/%version%/$VERSION/" kudo-e2e-test.yaml.tmpl > kudo-e2e-test.yaml
sed "s/%version%/$VERSION/" kudo-upgrade-test.yaml.tmpl > kudo-upgrade-test.yaml

# Download previous KUDO version for upgrade testing
if [[ "$(uname)" == "Darwin" ]]; then
    curl -L https://github.com/kudobuilder/kudo/releases/download/v${PREV_KUDO_VERSION}/kubectl-kudo_${PREV_KUDO_VERSION}_darwin_x86_64 --output bin/kubectl-oldkudo
elif [[ "$(expr substr $(uname -s) 1 5)" == "Linux" ]]; then
    curl -L https://github.com/kudobuilder/kudo/releases/download/v${PREV_KUDO_VERSION}/kubectl-kudo_${PREV_KUDO_VERSION}_linux_x86_64 --output bin/kubectl-oldkudo
fi

chmod +x bin/kubectl-oldkudo

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

    rm -rf operators
    git clone https://github.com/kudobuilder/operators
    mkdir operators/bin/
    cp ./bin/kubectl-kudo operators/bin/
    cp ./bin/manager operators/bin/
    cd operators && ./bin/kubectl-kudo test 2>&1 \
        | tee /dev/fd/2 \
        | go-junit-report -set-exit-code \
        > ../reports/kudo_operators_test_report.xml
else
    echo "Running E2E tests without junit output"

    ./bin/kubectl-kudo test --config kudo-e2e-test.yaml

    ./bin/kubectl-kudo test --config kudo-upgrade-test.yaml

    rm -rf operators
    git clone https://github.com/kudobuilder/operators
    mkdir operators/bin/
    cp ./bin/kubectl-kudo operators/bin/
    cp ./bin/manager operators/bin/
    cd operators && ./bin/kubectl-kudo test
fi
