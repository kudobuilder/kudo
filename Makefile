SHELL=/bin/bash -o pipefail
# Image URL to use all building/pushing image targets
DOCKER_TAG ?= $(shell git rev-parse HEAD)
DOCKER_IMG ?= kudobuilder/controller
EXECUTABLE := manager
CLI := kubectl-kudo
GIT_VERSION_PATH := github.com/kudobuilder/kudo/pkg/version.gitVersion
GIT_VERSION := $(shell git describe --abbrev=0 --tags --candidates=0 2>/dev/null || echo not-built-on-release)
GIT_COMMIT_PATH := github.com/kudobuilder/kudo/pkg/version.gitCommit
GIT_COMMIT := $(shell git rev-parse HEAD | cut -b -8)
SOURCE_DATE_EPOCH := $(shell git show -s --format=format:%ct HEAD)
BUILD_DATE_PATH := github.com/kudobuilder/kudo/pkg/version.buildDate
DATE_FMT := "%Y-%m-%dT%H:%M:%SZ"
BUILD_DATE := $(shell date -u -d "@$SOURCE_DATE_EPOCH" "+${DATE_FMT}" 2>/dev/null || date -u -r "${SOURCE_DATE_EPOCH}" "+${DATE_FMT}" 2>/dev/null || date -u "+${DATE_FMT}")
LDFLAGS := -X ${GIT_VERSION_PATH}=${GIT_VERSION:v%=%} -X ${GIT_COMMIT_PATH}=${GIT_COMMIT} -X ${BUILD_DATE_PATH}=${BUILD_DATE}
GOLANGCI_LINT_VER = "1.23.8"
SUPPORTED_PLATFORMS = amd64 arm64

export GO111MODULE=on

.PHONY: all
all: test manager

# Run unit tests
.PHONY: test
test:
ifdef _INTELLIJ_FORCE_SET_GOFLAGS
# Run tests from a Goland terminal. Goland already set '-mod=readonly'
	go test ./pkg/... ./cmd/... -v -coverprofile cover.out
else
	go test ./pkg/... ./cmd/... -v -mod=readonly -coverprofile cover.out
endif

# Run e2e tests
.PHONY: e2e-test
e2e-test: cli-fast manager-fast
	TEST_ONLY=$(TEST) ./hack/run-e2e-tests.sh

.PHONY: integration-test
# Run integration tests
integration-test: cli-fast manager-fast
	TEST_ONLY=$(TEST) ./hack/run-integration-tests.sh

.PHONY: operator-test
operator-test: cli-fast manager-fast
	./hack/run-operator-tests.sh

.PHONY: upgrade-test
upgrade-test: cli-fast manager-fast
	TEST_ONLY=$(TEST) ./hack/run-upgrade-tests.sh

.PHONY: test-clean
# Clean test reports
test-clean:
	rm -f cover.out cover-integration.out

.PHONY: lint
lint:
ifneq (${GOLANGCI_LINT_VER}, "$(shell golangci-lint --version 2>/dev/null | cut -b 27-32)")
	./hack/install-golangcilint.sh
endif
	golangci-lint --timeout 3m run

.PHONY: download
download:
	go mod download

.PHONY: prebuild
prebuild: generate lint


# Build manager binary
manager: prebuild manager-fast

.PHONY: manager-fast
# Build manager binary
manager-fast:
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
	go run -ldflags "${LDFLAGS}" ./cmd/manager

.PHONY: deploy
# Install KUDO into a cluster via kubectl kudo init
deploy:
	go run -ldflags "${LDFLAGS}" ./cmd/kubectl-kudo init

.PHONY: generate
# Generate code
generate:
ifneq ($(shell go list -f '{{.Version}}' -m sigs.k8s.io/controller-tools), $(shell controller-gen --version 2>/dev/null | cut -b 10-))
	@echo "(Re-)installing controller-gen. Current version:  $(controller-gen --version 2>/dev/null | cut -b 10-). Need $(go list -f '{{.Version}}' -m sigs.k8s.io/controller-tools)"
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@$$(go list -f '{{.Version}}' -m sigs.k8s.io/controller-tools)
endif
	controller-gen crd paths=./pkg/apis/... output:crd:dir=config/crds output:stdout
ifeq (, $(shell which go-bindata))
	go get github.com/go-bindata/go-bindata/go-bindata@$$(go list -f '{{.Version}}' -m github.com/go-bindata/go-bindata)
endif
	go-bindata -pkg crd -o pkg/kudoctl/kudoinit/crd/bindata.go -ignore README.md -nometadata config/crds
	./hack/update_codegen.sh

.PHONY: generate-clean
generate-clean:
	rm -rf ./hack/code-gen

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
clean:  cli-clean test-clean manager-clean

.PHONY: docker-build
# Build the docker image for each supported platform
docker-build: generate lint
	docker build --build-arg ldflags_arg="$(LDFLAGS)" -f Dockerfile -t $(DOCKER_IMG):$(DOCKER_TAG) .

.PHONY: imports
# used to update imports on project.  NOT a linter.
imports:
ifneq (${GOLANGCI_LINT_VER}, "$(shell golangci-lint --version 2>/dev/null | cut -b 27-32)")
	./hack/install-golangcilint.sh
endif
	golangci-lint run --disable-all -E goimports --fix

.PHONY: update-golden
# used to update the golden files present in ./pkg/.../testdata
# example: make update-golden
# tests in update==true mode show as failures
update-golden:
	go test ./pkg/... -v -mod=readonly --update=true

.PHONY: todo
# Show to-do items per file.
todo:
	@grep \
		--exclude-dir=hack \
		--exclude=Makefile \
		--exclude-dir=.git \
		--exclude-dir=bin \
		--text \
		--color \
		-nRo -E " *[^\.]TODO.*|SkipNow" .

# requires manifests. generate-manifests should be run first
# updating webhook requires ngrok to be running and updates the wh-config with the latest
# ngrok configuration.  ngrok config changes for each restart which is why this is a separate target
.PHONY: update-webhook-config
update-webhook-config:
	./hack/update-webhook-config.sh

# generates manifests from the kudo cli (kudo init) and captures the manifests needed for
# local development.  These are cached under the /hack/manifest-gen folder and are used to quickly
# start running and debugging kudo controller and webhook locally.
.PHONY: generate-manifests
generate-manifests:
	./hack/update-manifests.sh

# requires manifests. generate-manifests should be run first
# quickly sets a local dev env with the minimum necessary configurations to run
# the kudo manager locally.  after running dev-ready, it is possible to 'make run' or run from an editor for debugging.
# it currently does require ngrok.
.PHONY: dev-ready
dev-ready:
	./hack/deploy-dev-prereqs.sh
	./hack/update-webhook-config.sh
