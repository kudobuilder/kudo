# Build the manager binary
FROM golang:1.12 as builder

# Setting arguments
ARG git_version_arg
ARG git_commit_arg
ARG build_date_arg


# Copy in the go src
WORKDIR /go/src/github.com/kudobuilder/kudo
COPY pkg/    pkg/
COPY cmd/    cmd/
COPY go.mod go.mod
COPY go.sum go.sum
ENV GO111MODULE on

# Build with ldflags set
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager \
    -ldflags "-X ${git_version_arg} \
              -X ${git_commit_arg} \
              -X ${build_date_arg}" github.com/kudobuilder/kudo/cmd/manager

# Copy the controller-manager into a thin image
FROM ubuntu:18.04
WORKDIR /root/
COPY --from=builder /go/src/github.com/kudobuilder/kudo/manager .
ENTRYPOINT ["./manager"]
