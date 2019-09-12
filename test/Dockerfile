FROM kudobuilder/golang:1.13

WORKDIR $GOPATH/src/github.com/kudobuilder/kudo

# install docker
RUN apt-get update && apt-get install -y \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg2 \
    software-properties-common && curl -fsSL https://download.docker.com/linux/debian/gpg | apt-key add - && \
    add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/debian $(lsb_release -cs) stable" && \
    apt-get update && apt-get install -y docker-ce-cli

COPY Makefile go.mod go.sum ./
RUN make download
COPY config/ config/
COPY pkg/ pkg/
COPY cmd/ cmd/
COPY test/ test/
COPY kudo-test.yaml kudo-test.yaml

CMD make integration-test
