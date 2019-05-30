FROM kudobuilder/golang:1.12

WORKDIR $GOPATH/src/github.com/kudobuilder/kudo
COPY test/.git-credentials .git-credentials

COPY Makefile go.mod go.sum ./
RUN make download
COPY config/ config/
COPY pkg/ pkg/
COPY cmd/ cmd/

RUN cp .git-credentials ~/.git-credentials

ENTRYPOINT make test