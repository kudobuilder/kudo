---
title: Testing
types: docs
weight: 5
---

# Testing

KUDO uses a declarative integration testing harness for testing itself and Frameworks built on it. Test cases are written as plain Kubernetes resources and can be run against any Kubernetes cluster - allowing testing of Frameworks, KUDO, and any other Kubernetes resources or controllers.

## Table of Contents

* [Test harness usage](#test-harness-usage)
   * [Start a Kubernetes cluster with kind (optional)](#start-a-kubernetes-cluster-with-kind-optional)
   * [Run the Framework tests](#run-the-framework-tests)
* [Writing test cases](#writing-test-cases)
   * [Test case directory structure](#test-case-directory-structure)
   * [Test steps](#test-steps)
     * [Test assertions](#test-assertions)
       * [Advanced test assertions](#advanced-test-assertions)
* [Further Reading](#further-reading)

## Test harness usage

The Framework test suite is written and run by Framework developers and CI to test that Frameworks work correctly.

First, clone the [Frameworks repository](https://github.com/kudobuilder/frameworks):

```
git clone https://github.com/kudobuilder/frameworks.git
cd frameworks
```

Make sure that you have Go version 1.12 or greater and have Go modules enabled.

#### Start a Kubernetes cluster with kind (optional)

The Framework tests rely on a functioning Kubernetes cluster. As the test harness can run KUDO, it is not necessary to install KUDO prior to running the tests.

If you do not have a Kubernetes cluster to use for the tests, you can start one with [kind](https://github.com/kubernetes-sigs/kind):

There is a Make target in the [Frameworks repository](https://github.com/kudobuilder/frameworks) that you can use to start it:

```
make create-cluster
export KUBECONFIG=$(bin/kind get kubeconfig-path)
```

Now that the cluster is running, you can use it as a testing environment.

#### Run the Framework tests

To run the Framework test suite, run the following in the [Frameworks repository](https://github.com/kudobuilder/frameworks):

```
make test
```

Framework test suites are stored in the `tests` subdirectory of each [Framework](https://github.com/kudobuilder/frameworks/tree/master/repository), e.g.:

```
./repository/zookeeper/tests/
./repository/mysql/tests/
```

Every `Framework` and `FrameworkVersion` is installed into the cluster by `make test` prior to running the test suite.

## Writing test cases

To write a test case:

1. Create a directory for the test case. The directory should be created inside of the suite that the test case is a part of. For example, a Zookeeper Framework test case called `upgrade-test` would be created as `./repository/zookeeper/tests/upgrade-test/`.
2. Define the Kubernetes resources to apply and states to assert on in sequential test steps.

### Test case directory structure

Given the above Zookeeper test case called `upgrade-test`, an example test case directory structure with two test steps might look like:

```
./repository/zookeeper/tests/upgrade-test/00-instance.yaml
./repository/zookeeper/tests/upgrade-test/00-configmap.yaml
./repository/zookeeper/tests/upgrade-test/00-assert.yaml
./repository/zookeeper/tests/upgrade-test/01-upgrade.yaml
./repository/zookeeper/tests/upgrade-test/01-assert.yaml
```

Each file in the test case directory should start with a number indicating the index of the step. All files with the same index are a part of the same test step and applied simultaneously. Steps are applied serially in numerical order.

A file called `$index-assert.yaml` (where `$index` is the index of the test case) must also exist indicating the state that the test harness should wait for before continuing to the next step.

By default, the test harness will wait for up to 10 seconds for the assertion defined in the assertion file to be true before considering the test step failed.

In order for a test case to be successful, all steps must also complete successfully.

### Test steps

The test step files can contain any number of Kubernetes resources that should be applied as a part of the test step. Typically, this would be a KUDO `Instance` (see: [KUDO concepts](https://kudo.dev/docs/concepts/) or a `Deployment` or other resources required for the test, such as `Secrets` or test `Pods`.

Continuing with the upgrade-test example, create `00-instance.yaml`:

```
apiVersion: kudo.k8s.io/v1alpha1
kind: Instance
metadata:
  name: zk
spec:
  frameworkVersion:
    name: zookeeper-0.1.0
    namespace: default
    type: FrameworkVersions
  name: "zk"
  parameters:
    cpus: "0.3"
```

This test step will create a Zookeeper `Instance`. The namespace should not be specified in the resources as a namespace is created for the test case to run in.

#### Test assertions

Test assert files contain any number of Kubernetes resources that are expected to be created. Each resource must specify the `apiVersion`, `kind`, and `metadata`. The test harness watches each defined resource and waits for the state defined to match the state in Kubernetes. Once all resources have the correct state simultaneously, the test is considered successful.

Continuing with the `upgrade-test` example, create `00-instance.yaml`:

```
apiVersion: kudo.k8s.io/v1alpha1
kind: Instance
metadata:
  name: zk
status:
  status: COMPLETE
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: zk-zk
spec:
  template:
    spec:
     containers:
     - name: kubernetes-zookeeper
       resources:
         requests:
           memory: "1Gi"
           cpu: "300m"
status:
  readyReplicas: 3
```

This watches an `Instance` called `zk` to have its status set to `COMPLETE` and it expects a `StatefulSet` to also be created called `zk-zk` and it waits for all `Pods` in the `StatefulSet` to be ready.

##### Advanced test assertions

The test harness recognizes special `TestAssert` objects defined in the assert file. If present, they override default settings of the test assert.

```
apiVersion: kudo.k8s.io/v1alpha1
kind: TestAssert
timeout: 120
```

Options:

* `timeout`: the number of seconds to wait for the assertion to be true (default: `10`, type: `int`).

## Further Reading

* [Zookeeper Framework tests](https://github.com/kudobuilder/frameworks/tree/master/repository/zookeeper/tests)
* Design documentation for test harness: [KEP-0008 - Framework Testing](https://github.com/kudobuilder/kudo/blob/master/keps/0008-framework-testing.md)
* KUDO testing infrastructure and policies document: [KEP-0004 - Add Testing Infrastructure](https://github.com/kudobuilder/kudo/blob/master/keps/0004-add-testing-infrastructure.md)
