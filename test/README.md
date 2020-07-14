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

```bash
make integration-test
```

## Declarative tests

Most tests written for KUDO use the [declarative test harness](https://kudo.dev/docs/testing) with the controller-runtime's envtest (which starts `etcd` and `kube-apiserver` locally). This means that tests can be written for and run against KUDO without requiring a Kubernetes cluster (or even Docker).

The test suite is configured by `kudo-test.yaml` and the tests live in `test/integration/`. Prior to running the tests, all KUDO CRDs and manifests in `test/manifests/` are installed into the test cluster.

The test harness also starts KUDO, so it is recommended to use `go run` to run the test suite as this will include the latest built changes from your KUDO checkout.

There are multiple sets of tests using kuttl:

### KUTTL integration tests

These tests live in the `test/integration` directory and use the `test/kudo-integration-test.yaml` configuration file. These tests don't start KIND, they run the `etcd` and `kube-apiserver` binaries and therefore only support the most simple use cases. Things that can *not* be tested with these tests:

* Foreground deletion
* Creation of Pods (when a deployment is applied)
* Finalizers
* ...

```bash
make integration-test
```

If you want to run a single integration test, use:
```bash
make TEST=cli-test integration-test
```
where `cli-test` is the name of the `test/integration/cli-test` folder with the test files.

### End-to-End tests

These tests are located in `test/e2e` and use the `test/kudo-e2e-test.yaml.tmpl` configuration file. These tests spin up a single KIND cluster for all the tests and KUDO is installed as a part of the test harness. These tests can be used to ensure most of KUDO functionality but they have a bigger footprint than integration tests. 

To run all e2e tests:
```bash
make e2e-test
```

To run a single e2e-test
```bash
make TEST=plan-trigger e2e-test
```

### Upgrade tests

These tests are the heaviest ones. They live in `test/upgrade` and use the `test/kudo-upgrade-test.yaml.tmpl`. This suite spins up a single KIND cluster but does not install KUDO. Additionally, the tests are executed with a parallelism of one. It is therefore possible to install and uninstall KUDO in the tests, which is required for testing upgradability of KUDO.  

To run all upgrade tests:
```bash
make upgrade-test
```

To run a single upgrade-test
```bash
make TEST=upgrade-to-current upgrade-test
```


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

KUDO uses webhooks which require server certificates. By default, KUDO expects [cert-manager](https://cert-manager.io/) running in the cluster. There is also an option to use self-signed certificates by providing a `kudo init --unsafe-self-signed-webhook-ca` option during KUDO initialization. When developing and debugging KUDO locally, It is common to simply `make run` the controller in the terminal . However, since the API server running, in most cases inside the minikube, has be able to send POST requests to the webhook, additional configuration is necessary.

Here is what you need to make this setup work:

1. First of all the webhook needs access to the tls.crt and tls.key files, which by default is expected in `/tmp/cert` folder . You can (generate them using openssl)[https://www.digitalocean.com/community/tutorials/openssl-essentials-working-with-ssl-certificates-private-keys-and-csrs]:
```shell script
 ❯ openssl req -new -newkey rsa:4096 -x509 -sha256 -days 365 -nodes -out /tmp/cert/tls.crt -keyout /tmp/cert/tls.key -extensions san -config \
  <(echo "[req]"; 
    echo distinguished_name=req; 
    echo "[san]"; 
    echo subjectAltName=DNS:localhost,IP:127.0.0.1
    ) \
  -subj "/CN=*"
```
This command will fail if the `/tmp/cert` folder doesn't exist.  It is also possible to use the certificates that we use for the integration tests and change the default location for the certs when starting the manager using `KUDO_CERT_DIR` option:
```yaml
KUDO_CERT_DIR=./test/cert/ make run
```

2. Install ngrok: https://ngrok.com/ and run a local tunnel on the port 443 which will give you an url to your local machine:
```shell script
 ❯ ngrok http 443
  ...
  Forwarding                    https://ff6b2dd5.ngrok.io -> https://localhost:443
```
**note:** ngrok requires [registration]( https://ngrok.com/signup) in order to run against port 443.

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

**note:** The url used by ngrok changes for each run of ngrok.  It is possible to get the current url with `curl -s localhost:4040/api/tunnels | jq '.tunnels[] | select(.proto == "https") | .public_url'`

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
 ❯ curl -X POST "`curl -s localhost:4040/api/tunnels | jq '.tunnels[] | select(.proto == "https") | .public_url' -r`/admit-kudo-dev-v1beta1-instance"
{"response":{"uid":"","allowed":false,"status":{"metadata":{},"message":"contentType=, expected application/json","code":400}}}
```

**troubleshooting:** For a successful response to the test the `make run` controller needs to be running.  If you run ```curl -I -X POST "`curl -s localhost:4040/api/tunnels | jq '.tunnels[] | select(.proto == "https") | .public_url' -r`/admit-kudo-dev-v1beta1-instance"``` and get an error `502 Bad Gateway`, then the server isn't running or their is a configuration error.
