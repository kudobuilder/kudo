---
title: Asserts
type: docs
menu:
  docs:
    parent: 'Testing'
weight: 4
---

# Asserts

Test asserts are the part of a [test step](/docs/testing/steps) that define the state to wait for Kubernetes to reach. It is possible to match specific objects by name as well as match any object that matches a defined state.

* [Format](#format)
* [Getting a resource from the cluster](#getting-a-resource-from-the-cluster)
* [Listing resources in the cluster](#listing-resources-in-the-cluster)

# Format

The test assert file for a test step is found at `$index-assert.yaml`. So, if the test step index is `00`, the assert should be called `00-assert.yaml`. This file can contain any number of objects to match on. If the objects have a namespace set, it will be respected, but if a namespace is not set, then the test harness will look for the objects in the test case's namespace.

By default, a test step will wait for up to 30 seconds for the defined state to reached, see the [configuration reference](/docs/testing/reference#TestAssert) for documentation on configuring test asserts.

Note that an assertion file is optional, if it is not present, the test step will be considered successful immediately, once the objects in the test step have been created. It is also valid to create a test step that does not create any objects, but only has an assertion file.

# Getting a resource from the cluster

If an object has a name set, then the harness will look specifically for that object to exist and then verify that its state matches what is defined in the assert file. For example, if the assert file has:

```
apiVersion: v1
kind: Pod
metadata:
  name: my-pod
status:
  phase: Successful
```

Then the test harness will wait for the `my-pod` pod in the test namespace to have `status.phase=Successful`. Note that any fields *not* specified in the assert file will be ignored, making it possible to specify only the important fields for the test step.

# Listing resources in the cluster

If an object in the assert file has no name set, then the harness will list objects of that kind and expect there to be one that matches. For example, an assert:

```
apiVersion: v1
kind: Pod
status:
  phase: Successful
```

This example would wait for *any* pod to exist in the test namespace with the `status.phase=Successful`.
