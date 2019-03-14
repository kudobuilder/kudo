
# Image URL to use all building/pushing image targets
TAG ?= latest
IMG ?= kudobuilder/controller:${TAG}

all: test manager

deps:
	go install github.com/kudobuilder/kudo/vendor/github.com/golang/dep/cmd/dep
	dep check
	go install github.com/kudobuilder/kudo/vendor/golang.org/x/tools/cmd/goimports
	go install github.com/kudobuilder/kudo/vendor/golang.org/x/lint/golint

# Run tests
test: generate deps fmt vet lint imports manifests
	go test ./pkg/... ./cmd/... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager github.com/kudobuilder/kudo/cmd/manager

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet
	go run ./cmd/manager/main.go

# Install CRDs into a cluster
install: manifests
	kubectl apply -f config/crds

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	kubectl apply -f config/crds
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests:
	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go all

# Run go fmt against code
fmt:
	go fmt ./pkg/... ./cmd/...

# Run go vet against code
vet:
	go vet ./pkg/... ./cmd/...

# Run go lint against code
lint: 
	golint ./pkg/... ./cmd/...

# Run go imports against code
imports:
	goimports -w ./pkg/ ./cmd/

# Generate code
generate:
	go generate ./pkg/... ./cmd/...

# Build the docker image
docker-build: generate fmt vet manifests
	docker build . -t ${IMG}
	@echo "updating kustomize image patch file for manager resource"
	sed -i'' -e 's@image: .*@image: '"${IMG}"'@' ./config/default/manager_image_patch.yaml

# Push the docker image
docker-push:
	docker push ${IMG}
