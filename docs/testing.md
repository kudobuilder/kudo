---
title: Testing
types: docs
menu: docs
---

# Testing

KUDO uses a declarative integration testing harness for testing itself and Operators built on it. Test cases are written as plain Kubernetes resources and can be run against any Kubernetes cluster - allowing testing of Operators, KUDO, and any other Kubernetes resources or controllers.

## Table of Contents

* [Test harness usage](#test-harness-usage)
   * [Run the Operator tests](#run-the-operator-tests)
* [Writing test cases](#writing-test-cases)
   * [Test case directory structure](#test-case-directory-structure)
   * [Test steps](#test-steps)
     * [Deleting resources](#deleting-resources)
     * [Test assertions](#test-assertions)
       * [Listing objects](#listing-objects)
       * [Advanced test assertions](#advanced-test-assertions)
* [Further Reading](#further-reading)

## Test harness usage

The Operator test suite is written and run by Operator developers and CI to test that Operators work correctly.

First, clone the [Operators repository](https://github.com/kudobuilder/operators):

```
git clone https://github.com/kudobuilder/operators.git
cd operators
```

Make sure that you have the latest version of kudoctl installed.

#### Run the Operator tests

To run the Operator test suite, run the following in the [Operators repository](https://github.com/kudobuilder/operators):

```
kubectl kudo test
```

To run against a production Kubernetes cluster, run:

```
kubectl kudo test --start-kind=false
```

Operator test suites are stored in the `tests` subdirectory of each [Operator](https://github.com/kudobuilder/operators/tree/master/repository), e.g.:

```
./repository/zookeeper/tests/
./repository/mysql/tests/
```

Every `Operator` and `OperatorVersion` is installed into the cluster by `make test` prior to running the test suite.

## Writing test cases

To write a test case:

1. Create a directory for the test case. The directory should be created inside of the suite that the test case is a part of. For example, a Zookeeper Operator test case called `upgrade-test` would be created as `./repository/zookeeper/tests/upgrade-test/`.
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
apiVersion: kudo.dev/v1alpha1
kind: Instance
metadata:
  name: zk
spec:
  operatorVersion:
    name: zookeeper-0.1.0
    namespace: default
    type: OperatorVersions
  name: "zk"
  parameters:
    cpus: "0.3"
```

This test step will create a Zookeeper `Instance`. The namespace should not be specified in the resources as a namespace is created for the test case to run in.

#### Deleting resources

It is possible to delete existing resources at the beginning of a test step. Create a `TestStep` object in your step to configure it:

```
apiVersion: kudo.dev/v1alpha1
kind: TestStep
delete:
- name: my-pod
  kind: Pod
  apiVersion: v1
- kind: Pod
  apiVersion: v1
  labels:
    app: nginx
- kind: Pod
  apiVersion: v1
```

The test harness will delete for each resource referenced in the delete list and wait for them to disappear from the API. If the object fails to delete, the test step will fail.

In the first delete example, the `Pod` called `my-pod` will be deleted. In the second, all `Pods` matching the `app=nginx` label will be deleted. In the third example, all pods in the namespace would be deleted.

#### Test assertions

Test assert files contain any number of Kubernetes resources that are expected to be created. Each resource must specify the `apiVersion`, `kind`, and `metadata`. The test harness watches each defined resource and waits for the state defined to match the state in Kubernetes. Once all resources have the correct state simultaneously, the test is considered successful.

Continuing with the `upgrade-test` example, create `00-instance.yaml`:

```
apiVersion: kudo.dev/v1alpha1
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

##### Listing objects

If the object `name` is omitted from the object metadata, it is possible to list objects and verify that one of them matches the desired state. This can be useful, for example, to check the `Pods` created by a `Deployment`.

```
apiVersion: v1
kind: Pod
metadata:
  labels:
    app: nginx
status:
  phase: Running
```

This would verify that a pod with the `app=nginx` label is running.

##### Advanced test assertions

The test harness recognizes special `TestAssert` objects defined in the assert file. If present, they override default settings of the test assert.

```
apiVersion: kudo.dev/v1alpha1
kind: TestAssert
timeout: 120
```

Options:

* `timeout`: the number of seconds to wait for the assertion to be true (default: `10`, type: `int`).

## Further Reading

* [Zookeeper Operator tests](https://github.com/kudobuilder/operators/tree/master/repository/zookeeper/tests)
* Design documentation for test harness: [KEP-0008 - Operator Testing](https://github.com/kudobuilder/kudo/blob/master/keps/0008-operator-testing.md)
* KUDO testing infrastructure and policies document: [KEP-0004 - Add Testing Infrastructure](https://github.com/kudobuilder/kudo/blob/master/keps/0004-add-testing-infrastructure.md)
