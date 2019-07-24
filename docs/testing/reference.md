---
title: Configuration Reference
type: docs
menu:
  docs:
    parent: 'Testing'
weight: 6
---

# Configuration Reference

* [TestSuite](#testsuite)
* [TestStep](#teststep)
* [TestAssert](#testassert)

# TestSuite

The `TestSuite` object specifies the settings for the entire test suite and should live in the test suite configuration file (`kudo-test.yaml` by default, or `--config`):

```
apiVersion: kudo.dev/v1alpha1
kind: TestSuite
startKIND: true
startKUDO: true
testDirs:
- tests/e2e/
timeout: 120
```

Supported settings:

Field             |      Type       | Description                                                                              | Default
------------------|-----------------|------------------------------------------------------------------------------------------|--------
crdDir            | string          | Path to CRDs to install before running tests.                                            | 
manifestDirs      | list of strings | Paths to manifests to install before running tests.                                      | 
testDirs          | list of strings | Directories containing test cases to run.                                                | 
startControlPlane | bool            | Whether or not to start a local etcd and kubernetes API server for the tests.            | false
startKIND         | bool            | Whether or not to start a local kind cluster for the tests.                              | false
kindConfig        | string          | Path to the KIND configuration file to use.                                              | 
kindContext       | string          | KIND context to use.                                                                     | "kind"
startKUDO         | bool            | Whether or not to start the KUDO controller for the tests.                               | false
skipDelete        | bool            | If set, do not delete the resources after running the tests (implies SkipClusterDelete). | false
skipClusterDelete | bool            | If set, do not delete the mocked control plane or kind cluster.                          | false
timeout           | int             | Override the default timeout of 30 seconds (in seconds).                                 | 30
parallel          | int             | The maximum number of tests to run at once.                                              | 8
artifactsDir      | string          | The directory to output artifacts to (current working directory if not specified).       | .

# TestStep

The `TestStep` object can be used to specify settings for a test step and can be specified in any test step YAML.

```
apiVersion: kudo.dev/v1alpha1
kind: TestStep
delete:
- apiVersion: v1
  kind: Pod
  name: my-pod
```

Supported settings:

Field   |          Type             | Description
--------|---------------------------|---------------------------------------------------------------------
delete  | list of object references | A list of objects to delete, if they do not already exist, at the beginning of the test step. The test harness will wait for the objects to be successfully deleted before applying the objects in the step.
index   | int                       | Override the test step's index.

Object Reference:

Field      |   Type | Description
-----------|--------|---------------------------------------------------------------------
apiVersion | string | The Kubernetes API version of the objects to delete.
kind       | string | The Kubernetes kind of the objects to delete.
name       | string | If specified, the name of the object to delete. If not specified, all objects that match the specified labels will be deleted.
namespace  | string | The namespace of the objects to delete.
labels     | map    | If specified, a label selector to use when looking up objects to delete. If both labels and name are unspecified, then all resources of the specified kind in the namespace will be deleted.


# TestAssert

The `TestAssert` object can be used to specify settings for a test step's assert and must be specified in the test step's assert YAML.

```
apiVersion: kudo.dev/v1alpha1
kind: TestAssert
timeout: 30
```

Supported settings:

Field   | Type | Description                                           | Default
--------|------|-------------------------------------------------------|-------------
timeout | int  | Number of seconds that the test is allowed to run for | 30
