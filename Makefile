SHELL=/bin/bash -o pipefail
# Image URL to use all building/pushing image targets
DOCKER_TAG ?= $(shell git rev-parse HEAD)
DOCKER_IMG ?= kudobuilder/controller
EXECUTABLE := manager
CLI := kubectl-kudo
GIT_VERSION_PATH := github.com/kudobuilder/kudo/pkg/version.gitVersion
GIT_VERSION := $(shell git describe --abbrev=0 --tags | cut -b 2-)
GIT_COMMIT_PATH := github.com/kudobuilder/kudo/pkg/version.gitCommit
GIT_COMMIT := $(shell git rev-parse HEAD | cut -b -8)
SOURCE_DATE_EPOCH := $(shell git show -s --format=format:%ct HEAD)
BUILD_DATE_PATH := github.com/kudobuilder/kudo/pkg/version.buildDate
DATE_FMT := "%Y-%m-%dT%H:%M:%SZ"
BUILD_DATE := $(shell date -u -d "@$SOURCE_DATE_EPOCH" "+${DATE_FMT}" 2>/dev/null || date -u -r "${SOURCE_DATE_EPOCH}" "+${DATE_FMT}" 2>/dev/null || date -u "+${DATE_FMT}")
LDFLAGS := -X ${GIT_VERSION_PATH}=${GIT_VERSION} -X ${GIT_COMMIT_PATH}=${GIT_COMMIT} -X ${BUILD_DATE_PATH}=${BUILD_DATE}
ENABLE_WEBHOOKS ?= false

export GO111MODULE=on

.PHONY: all
all: test manager

.PHONY: test
# Run tests
test:
	go test ./pkg/... ./cmd/... -v -mod=readonly -coverprofile cover.out

# Run e2e tests
.PHONY: e2e-test
e2e-test: cli-fast
	./hack/run-e2e-tests.sh

.PHONY: integration-test
# Run integration tests
integration-test: cli-fast
	./hack/run-integration-tests.sh

.PHONY: test-clean
# Clean test reports
test-clean:
	rm -f cover.out cover-integration.out

.PHONY: lint
lint:
ifeq (, $(shell which golangci-lint))
	./hack/install-golangcilint.sh
endif
	golangci-lint run

.PHONY: download
download:
	go mod download

.PHONY: prebuild
prebuild: generate lint

.PHONY: manager
# Build manager binary
manager: prebuild
	# developer convenience for platform they are running
	go build -ldflags "${LDFLAGS}" -o bin/$(EXECUTABLE) github.com/kudobuilder/kudo/cmd/manager

.PHONY: manager-clean
# Clean manager build
manager-clean:
	rm -f bin/manager

.PHONY: run
# Run against the configured Kubernetes cluster in ~/.kube/config
run:
    # for local development, webhooks are disabled by default
    # if you enable them, you have to take care of providing the TLS certs locally
	ENABLE_WEBHOOKS=${ENABLE_WEBHOOKS} go run -ldflags "${LDFLAGS}" ./cmd/manager

.PHONY: deploy
# Install KUDO into a cluster via kubectl kudo init
deploy:
	go run -ldflags "${LDFLAGS}" ./cmd/kubectl-kudo init

.PHONY: deploy-clean
deploy-clean:
	go run ./cmd/kubectl-kudo  init --dry-run --output yaml | kubectl delete -f -

.PHONY: generate
# Generate code
generate:
ifeq (, $(shell which go-bindata))
	go get github.com/go-bindata/go-bindata/go-bindata
endif
	go-bindata -pkg crd -o pkg/kudoctl/kudoinit/crd/bindata.go -ignore README.md config/crds
	./hack/update_codegen.sh

.PHONY: generate-clean
generate-clean:
	rm -rf hack/code-gen

.PHONY: cli-fast
# Build CLI but don't lint or run code generation first.
cli-fast:
	go build -ldflags "${LDFLAGS}" -o bin/${CLI} ./cmd/kubectl-kudo

.PHONY: cli
# Build CLI
cli: prebuild cli-fast

.PHONY: cli-clean
# Clean CLI build
cli-clean:
	rm -f bin/${CLI}

# Install CLI
cli-install:
	go install -ldflags "${LDFLAGS}" ./cmd/kubectl-kudo

.PHONY: clean
# Clean all
clean:  cli-clean test-clean manager-clean deploy-clean

.PHONY: docker-build
# Build the docker image
docker-build: generate lint
	docker build --build-arg git_version_arg=${GIT_VERSION_PATH}=v${GIT_VERSION} \
	--build-arg git_commit_arg=${GIT_COMMIT_PATH}=${GIT_COMMIT} \
	--build-arg build_date_arg=${BUILD_DATE_PATH}=${BUILD_DATE} . -t ${DOCKER_IMG}:${DOCKER_TAG}
	docker tag ${DOCKER_IMG}:${DOCKER_TAG} ${DOCKER_IMG}:v${GIT_VERSION}
	docker tag ${DOCKER_IMG}:${DOCKER_TAG} ${DOCKER_IMG}:latest

.PHONY: docker-push
# Push the docker image
docker-push:
	docker push ${DOCKER_IMG}:${DOCKER_TAG}
	docker push ${DOCKER_IMG}:${GIT_VERSION}
	docker push ${DOCKER_IMG}:latest

.PHONY: imports
# used to update imports on project.  NOT a linter.
imports:
ifeq (, $(shell which golangci-lint))
	./hack/install-golangcilint.sh
endif
	golangci-lint run --disable-all -E goimports --fix

.PHONY: todo
# Show to-do items per file.
todo:
	@grep \
		--exclude-dir=hack \
		--exclude=Makefile \
		--text \
		--color \
		-nRo -E ' TODO:.*|SkipNow' .
