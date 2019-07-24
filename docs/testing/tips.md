---
title: Tips and Tricks
type: docs
menu:
  docs:
    parent: 'Testing'
weight: 5
---

# Tips and Tricks

This document contains some tips and gotchas that can be helpful when writing tests.

* [Kubernetes Events](#kubernetes-events)
* [Custom Resource Definitions](#custom-resource-definitions)

# Kubernetes Events

Kubernetes events are regular Kubernetes objects and can be asserted on just like any other object:

```
apiVersion: v1
kind: Event
reason: Started
source:
  component: kubelet
involvedObject:
  apiVersion: v1
  kind: Pod
  name: my-pod
```

# Custom Resource Definitions

New Custom Resource Definitions are not immediately available for use in the Kubernetes API until the Kubernetes API has acknowledged them. 

If a Custom Resource Definition is being defined inside of a test step, be sure to to wait for the `CustomResourceDefinition` object to appear.

For example, given this Custom Resource Definition in `tests/e2e/crd-test/00-crd.yaml`:

```
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: mycrds.mycrd.k8s.io
spec:
  group: mycrd.k8s.io
  version: v1alpha1
  names:
    kind: MyCRD
    listKind: MyCRDList
    plural: mycrds
    singular: mycrd
  scope: Namespaced
```

Create the following assert `tests/e2e/crd-test/00-assert.yaml`:

```
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: mycrds.mycrd.k8s.io
status:
  acceptedNames:
    kind: MyCRD
    listKind: MyCRDList
    plural: mycrds
    singular: mycrd
  storedVersions:
  - v1alpha1
```

And then the CRD can be used in subsequent steps, `tests/e2e/crd-test/01-use.yaml`:

```
apiVersion: mycrd.k8s.io/v1alpha1
kind: MyCRD
spec:
  test: test
```

Note that CRDs created via the `crdDir` test suite configuration are available for use immediately and do not require an assert like this.
