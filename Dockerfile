# Build the manager binary
FROM golang:1.11.2 as builder

# Copy in the go src
WORKDIR /go/src/github.com/kubernetes-sigs/maestro
COPY pkg/    pkg/
COPY cmd/    cmd/
COPY vendor/ vendor/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager github.com/kubernetes-sigs/maestro/cmd/manager

# Copy the controller-manager into a thin image
FROM ubuntu:latest
WORKDIR /root/
COPY --from=builder /go/src/github.com/kubernetes-sigs/maestro/manager .
ENTRYPOINT ["./manager"]
