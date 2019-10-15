#!/usr/bin/env bash

if [ "INTEGRATION_OUTPUT_JUNIT" = "true" ]
then
    mkdir -p reports/
    go get github.com/jstemmer/go-junit-report
    go test -tags integration ./pkg/... ./cmd/... -v -mod=readonly -coverprofile cover-integration.out 2>&1 |tee /dev/fd/2 |go-junit-report -set-exit-code > reports/integration_report.xml
    go run ./cmd/kubectl-kudo test 2>&1 |tee /dev/fd/2 |go-junit-report -set-exit-code > reports/kudo_test_report.xml
else
    go test -tags integration ./pkg/... ./cmd/... -v -mod=readonly -coverprofile cover-integration.out
    go run ./cmd/kubectl-kudo test
fi