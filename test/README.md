# Setup

To setup the tests locally, you need either:

* Docker

Or:

* Go
* kubebuilder

## Downloading kubebuilder

To setup kubebuilder, fetch the latest release from [Github](https://github.com/kubernetes-sigs/kubebuilder/releases) and extract `etcd` and `kube-apiserver` into `/usr/local/kubebuilder/bin/`.

# Docker only

If you don't want to install kubebuilder and other dependencies of KUDO locally, you can build KUDO and run the tests inside a Docker container.

To run tests inside a Docker container, you can just execute:

`./test/run_tests.sh`


# Without Docker

## Running unit tests

Unit tests are written for KUDO using the standard Go testing library. You can run the unit tests:

```
make test
```

## Running integration tests

Or run all tests:

```
make integration-test
```

## Declarative tests

Most tests written for KUDO use the [declarative test harness](https://kudo.dev/docs/testing) with the controller-runtime's envtest (which starts `etcd` and `kube-apiserver` locally). This means that tests can be written for and run against KUDO without requiring a Kubernetes cluster (or even Docker).

The test suite is configured by `kudo-test.yaml` and the tests live in `test/integration/`. Prior to running the tests, all KUDO CRDs and manifests in `test/manifests/` are installed into the test cluster.

The test harness also starts KUDO, so it is recommended to use `go run` to run the test suite as this will include the latest built changes from your KUDO checkout.

### CLI examples

Run all integration tests:

```
go run ./cmd/kubectl-kudo test
```

Run a specific integration test (e.g., the `patch` test from `test/integration/patch`):

```
go run ./cmd/kubectl-kudo test --test patch
```

Run tests against a live cluster:

```
go run ./cmd/kubectl-kudo test --start-control-plane=false
```

Run tests against a live cluster and do not delete resources after running:

```
go run ./cmd/kubectl-kudo test --start-control-plane=false --skip-delete
```
### Update golden files of the directory testdata

You can update golden files across the project by running `make update-golden`

Examples:
```
make update-golden
```

Running a non-existing test would return a
```
ok      github.com/kudobuilder/kudo/pkg/kudoctl/cmd     0.048s [no tests to run]
```

This will update all golden files.   There is no fear in updating the entire project as any change resulting in a golden file test failure would need to be updated regardless.
