FROM kudobuilder/golang:1.12

WORKDIR $GOPATH/src/github.com/kudobuilder/kudo

COPY Makefile go.mod go.sum ./
RUN make download
COPY config/ config/
COPY pkg/ pkg/
COPY cmd/ cmd/
COPY kudo-test.yaml kudo-test.yaml

ENTRYPOINT make integration-test
