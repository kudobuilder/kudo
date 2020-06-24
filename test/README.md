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


## How to set up a webhook locally

KUDO uses webhooks that themselves require server certificates. By default, KUDO expects [cert-manager](https://cert-manager.io/) running in the cluster. There is also an option to use self-signed certificates by providing a `kudo inint --unsafe-self-signed-webhook-ca` option during KUDO initialization. It is also common to simply `make run` the controller in the console when developing and debugging KUDO locally. However, since the API server running, in most cases inside the minikube, has be able to send POST requests to the webhook, this setup doesn't work out of the box. 

Here what you need to make this setup work:

1. First of all the webhook needs tls.crt and tls.key files in /tmp/cert to start. You can use openssl to generate them:
```shell script
 ❯ openssl req -new -newkey rsa:4096 -x509 -sha256 -days 365 -nodes -out /tmp/cert/tls.crt -keyout /tmp/cert/tls.key
```
and generate the certificates in the default location `/tmp/cert` or use the certificates that we use of the integration 
tests and start the manager using `KUDO_CERT_DIR` option:
```yaml
KUDO_CERT_DIR=./test/cert/ make run
```

2. Install ngrok: https://ngrok.com/ and run a local tunnel on the port 443 which will give you an url to your local machine:
```shell script
 ❯ ngrok http 443
  ...
  Forwarding                    https://ff6b2dd5.ngrok.io -> https://localhost:443
```

3. Set webhooks[].clientConfig.url to the url of the above tunnel and apply/edit webhook configuration to the cluster:
```yaml
---
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: kudo-manager-instance-admission-webhook-config
webhooks:
- admissionReviewVersions:
  - v1beta1
  clientConfig:
    url: https://ff6b2dd5.ngrok.io/admit-kudo-dev-v1beta1-instance
  failurePolicy: Fail
  matchPolicy: Equivalent
  name: instance-admission.kudo.dev
  namespaceSelector: {}
  objectSelector: {}
  reinvocationPolicy: Never
  rules:
  - apiGroups:
    - kudo.dev
    apiVersions:
    - v1beta1
    operations:
    - CREATE
    - UPDATE
    resources:
    - instances
    scope: Namespaced
  sideEffects: None
  timeoutSeconds: 30
---
```

The difference between this one and the one generate by the `kudo init --unsafe-self-signed-webhook-ca` command, (see this [method](pkg/kudoctl/kudoinit/prereq/webhook.go:163) for more information) is the usage of `webhooks[].clientConfig.url` (which points to our ngrok-tunnel) instead of `webhooks[].clientConfig.Service`.

4. Finally, you can run your local manager:
```shell script
 ❯ make run
```

and if everything was setup correctly the log should show:
```text
...
 Setting up webhooks
 Instance admission webhook
 Done! Everything is setup, starting KUDO manager now
```

5. Test the webhook with:
```shell script
 ❯ curl -X POST https://ff6b2dd5.ngrok.io/admit-kudo-dev-v1beta1-instance
{"response":{"uid":"","allowed":false,"status":{"metadata":{},"message":"contentType=, expected application/json","code":400}}}
```