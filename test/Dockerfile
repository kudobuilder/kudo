FROM kudobuilder/golang:1.13

WORKDIR $GOPATH/src/github.com/kudobuilder/kudo

COPY Makefile go.mod go.sum ./
RUN make download
COPY config/ config/
COPY pkg/ pkg/
COPY cmd/ cmd/
COPY hack/run-integration-tests.sh hack/run-integration-tests.sh
COPY test/ test/
COPY kudo-test.yaml kudo-test.yaml

ENTRYPOINT make integration-test
