---
title: Testing
type: docs
menu: docs
---

# KUDO Test Harness

KUDO includes a declarative integration testing harness for testing Operators, KUDO, and any other Kubernetes applications or controllers. Test cases are written as plain Kubernetes resources and can be run against a mocked control plane, locally in KIND, or any other Kubernetes cluster.

Whether you are developing an application, controller, operator, or deploying Kubernetes clusters the KUDO test harness helps you easily write portable end-to-end, integration, and conformance tests for Kubernetes without needing to write any code.

* [Installation](#installation)
* [Writing your first test](#writing-your-first-test)
   * [Create a test case](#create-a-test-case)
   * [Run the tests](#run-the-tests)
   * [Write a second test step](#write-a-second-test-step)
   * [Test suite configuration](#test-suite-configuration)

# Installation

The test harness CLI is included in the KUDO CLI, to install we can install the CLI using [krew](https://github.com/kubernetes-sigs/krew):

```
krew install kudo
```

You can now invoke the kudo test CLI:

```
kubectl kudo test --help
```

See the [KUDO installation guide](/docs/cli#install) for alternative installation methods.

# Writing your first test

Now that the kudo CLI is installed, we can write a test. The KUDO test CLI organizes tests into suites:

* A "test step" defines a set of Kubernetes manifests to apply and a state to assert on (wait for or expect).
* A "test case" is a collection of test steps that are run serially - if any test step fails then the entire test case is considered failed.
* A "test suite" is comprised of many test cases that are run in parallel.
* The "test harness" is the tool that runs test suites (the KUDO CLI).

## Create a test case

First, let's create a directory for our test suite, let's call it `tests/e2e`:

```
mkdir -p tests/e2e
```

Next, we'll create a directory for our test case, the test case will be called `example-test`:

```
mkdir tests/e2e/example-test
```

Inside of `tests/e2e/example-test/` create our first test step, `00-install.yaml`, which will create a deployment called `example-deployment`:

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-deployment
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:latest
        ports:
        - containerPort: 80
```

Note that in this example, the Deployment does not have a `namespace` set. The test harness will create a namespace for each test case and run all of the test steps inside of it. However, if a resource already has a namespace set (or is not a namespaced resource), then the harness will respect the namespace that is set.

Each filename in the test case directory should start with an index (in this example `00`) that indicates which test step the file is a part of. Files that do not start with a step index are ignored and can be used for documentation or other test data. Test steps are run in order and each must be successful for the test case to be considered successful.

Now that we have a test step, we need to create a test assert. The assert's filename should be the test step index followed by `-assert.yaml`. Create `tests/e2e/example-test/00-assert.yaml`:

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-deployment
status:
  readyReplicas: 3
```

This test step will be considered completed once the pod matches the state that we have defined. If the state is not reached by the time the assert's timeout has expired (30 seconds, by default), then the test step and case will be considered failed.

## Run the tests

Let's run this test suite:

```
kubectl kudo test --start-kind=true ./tests/e2e/
```

Running this command will:

* Start a [kind (kubernetes-in-docker) cluster](https://github.com/kubernetes-sigs/kind), if there is not already one running.
* Create a new namespace for the test case.
* Create the resources defined in `tests/e2e/example-test/00-install.yaml`.
* Wait for the state defined in `tests/e2e/example-test/00-assert.yaml` to be reached.
* Collect the kind cluster's logs.
* Tear down the kind cluster (or you can run kudo test with `--skip-cluster-delete` to keep the cluster around after the tests run).

## Write a second test step

Now that we have successfully written a test case, let's add another step to it. In this step, let's increase the number of replicas on the deployment we created in the first step from 3 to 4.

Create `tests/e2e/example-test/01-scale.yaml`:

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-deployment
spec:
  replicas: 4
```

Now create an assert for it in `tests/e2e/example-test/01-assert.yaml`:

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-deployment
status:
  readyReplicas: 4
```

Run the test suite again and the test will pass:

```
kubectl kudo test --start-kind=true ./tests/e2e/
```

## Test suite configuration

To add this test suite to your project, create a `kudo-test.yaml` file:

```
apiVersion: kudo.dev/v1alpha1
kind: TestSuite
testDirs:
- ./tests/e2e/
startKIND: true
```

Now we can run the tests just by running `kubectl kudo test` with no arguments.

Any arguments provided on the command line will override the settings in the `kudo-test.yaml` file, e.g. to skip using kind and run the tests against a live Kubernetes cluster, run:

```
kubectl kudo test --start-kind=false
```

Now that your first test suite is configured, see [test environments](/docs/testing/test-environments) for documentation on customizing your test environment or the [test step documentation](/docs/testing/steps) to write more advanced tests.
