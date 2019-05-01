
# Image URL to use all building/pushing image targets
DOCKER_TAG ?= $(shell git rev-parse HEAD)
DOCKER_IMG ?= kudobuilder/controller
EXECUTABLE := manager
CLI := kubectl-kudo
GIT_VERSION_PATH := github.com/kudobuilder/kudo/pkg/version.gitVersion
GIT_VERSION := $(shell git describe --abbrev=0 --tags)
GIT_COMMIT_PATH := github.com/kudobuilder/kudo/pkg/version.gitCommit
GIT_COMMIT := $(shell git rev-parse HEAD)
SOURCE_DATE_EPOCH := $(shell git show -s --format=format:%ct HEAD)
BUILD_DATE_PATH := github.com/kudobuilder/kudo/pkg/version.buildDate
BUILD_DATE := $(shell date -r ${SOURCE_DATE_EPOCH} '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -X ${GIT_VERSION_PATH}=${GIT_VERSION} -X ${GIT_COMMIT_PATH}=${GIT_COMMIT} -X ${BUILD_DATE_PATH}=${BUILD_DATE}

export GO111MODULE=on

.PHONY: all
all: test manager

.PHONY: test
# Run tests
test:
	go test ./pkg/... ./cmd/... -mod=readonly -coverprofile cover.out

.PHONY: test-clean
# Clean test reports
test-clean:
	rm -f cover.out

.PHONY: check-formatting
check-formatting: vet lint staticcheck
	./hack/check_formatting.sh

.PHONY: download
download:
	go mod download

.PHONY: prebuild
prebuild: generate check-formatting

.PHONY: manager
# Build manager binary
manager: prebuild
	# developer convince for platform they are running
	go build -ldflags "${LDFLAGS}" -o bin/$(EXECUTABLE) github.com/kudobuilder/kudo/cmd/manager

	# platforms for distribution
	GOARCH=amd64 GOOS=darwin go build -ldflags "${LDFLAGS}" -o bin/darwin/amd64/$(EXECUTABLE) github.com/kudobuilder/kudo/cmd/manager
	GOARCH=amd64 GOOS=linux go build -ldflags "${LDFLAGS}" -o bin/linux/amd64/$(EXECUTABLE) github.com/kudobuilder/kudo/cmd/manager
	GOARCH=amd64 GOOS=windows go build -ldflags "${LDFLAGS}"-o bin/windows/amd64/$(EXECUTABLE) github.com/kudobuilder/kudo/cmd/manager

.PHONY: manager-clean
# Clean manager build
manager-clean:
	rm -f bin/manager
	rm -rf bin/darwin/amd64/$(EXECUTABLE)
	rm -rf bin/linux/amd64/$(EXECUTABLE)
	rm -rf bin/windows/amd64/$(EXECUTABLE)

.PHONY: run
# Run against the configured Kubernetes cluster in ~/.kube/config
run:
	go run -ldflags "${LDFLAGS}" ./cmd/manager/main.go

.PHONY: install-crds
install-crds:
	kubectl apply -f config/crds

.PHONY: deploy
# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	kubectl apply -f config/crds
	kustomize build config/default | kubectl apply -f -

.PHONY: deploy-clean
deploy-clean:
	kubectl delete -f config/crds
	# kustomize build config/default | kubectl delete -f -

.PHONY: manifests
# Generate manifests e.g. CRD, RBAC etc.
manifests:
	./hack/update_manifests.sh

.PHONY: manifests-clean
manifests-clean:
	rm -rf hack/controller-gen

.PHONY: fmt
# Run go fmt against code
fmt:
	go fmt ./pkg/... ./cmd/...

.PHONY: vet
# Run go vet against code
vet:
	go vet ./pkg/... ./cmd/...

.PHONY: lint
# Run go lint against code
lint:
	go install golang.org/x/lint/golint
	golint ./pkg/... ./cmd/...

.PHONY: staticcheck
# Runs static check
staticcheck:
	go install honnef.co/go/tools/cmd/staticcheck
	staticcheck ./...

.PHONY: imports
# Run go imports against code
imports:
	go install golang.org/x/tools/cmd/goimports
	goimports -w ./pkg/ ./cmd/

.PHONY: generate
# Generate code
generate:
	./hack/update_codegen.sh

.PHONY: generate-clean
generate-clean:
	rm -rf hack/code-gen

.PHONY: cli
# Build CLI
cli: prebuild
	# developer convince for platform they are running
	go build -ldflags "${LDFLAGS}" -o bin/${CLI} cmd/kubectl-kudo/main.go

	# platforms for distribution
	GOARCH=amd64 GOOS=darwin go build -ldflags "${LDFLAGS}" -o bin/darwin/amd64/${CLI} cmd/kubectl-kudo/main.go
	GOARCH=amd64 GOOS=linux go build -ldflags "${LDFLAGS}" -o bin/linux/amd64/${CLI} cmd/kubectl-kudo/main.go
	GOARCH=amd64 GOOS=windows go build -ldflags "${LDFLAGS}" -o bin/windows/${CLI} cmd/kubectl-kudo/main.go

.PHONY: cli-clean
# Clean CLI build
cli-clean:
	rm -f bin/${CLI}
	rm -rf bin/darwin/amd64/${CLI}
	rm -rf bin/linux/amd64/${CLI}
	rm -rf bin/windows/${CLI}

# Install CLI
cli-install:
	go install -ldflags "${LDFLAGS}" ./cmd/kubectl-kudo

.PHONY: clean
# Clean all
clean:  cli-clean test-clean manager-clean deploy-clean
	rm -rf bin/darwin
	rm -rf bin/linux
	rm -rf bin/windows

.PHONY: docker-build
# Build the docker image
docker-build: generate check-formatting manifests
	docker build --build-arg git_version_arg=${GIT_VERSION_PATH}=${GIT_VERSION} \
	--build-arg git_commit_arg=${GIT_COMMIT_PATH}=${GIT_COMMIT} \
	--build-arg build_date_arg=${BUILD_DATE_PATH}=${BUILD_DATE} . -t ${DOCKER_IMG}:${DOCKER_TAG}
	docker tag ${DOCKER_IMG}:${DOCKER_TAG} ${DOCKER_IMG}:${GIT_VERSION}
	docker tag ${DOCKER_IMG}:${DOCKER_TAG} ${DOCKER_IMG}:latest
	@echo "updating kustomize image patch file for manager resource"
	sed -i'' -e 's@image: .*@image: '"${DOCKER_IMG}:${GIT_VERSION}"'@' ./config/default/manager_image_patch.yaml

.PHONY: docker-push
# Push the docker image
docker-push:
	docker push ${DOCKER_IMG}:${DOCKER_TAG}
	docker push ${DOCKER_IMG}:${GIT_VERSION}
	docker push ${DOCKER_IMG}:latest
