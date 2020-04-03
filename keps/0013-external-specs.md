---
kep-number: 0013
title: External Specs
short-desc: Run non-KUDO defined applications as Operators
authors:
  - "@runyontr"
owners:
  - @runyontr
  - "@gerred"
creation-date: 2019-06-18
last-updated: 2019-06-18
status: provisional
---

# External Application Specs

## Table of Contents

- [Table of Contents](#table-of-contents)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories [optional]](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
- [Infrastructure Needed [optional]](#infrastructure-needed-optional)

## Summary

Organizations and companies already spend time building, updating and debugging specs for deploying their applications. Asking them to spend additional time to maintain a KUDO spec may prevent adoption of the platform. Improving KUDO to be able to consume more than just a KUDO Spec defined in [KEP-0009](0009-operator-toolkit.md) would allow for oragnizations to provide access to their applications to the KUDO community.

## Motivation

While being able to package our own application definition via [KEP0009](0009-operator-toolkit.md), being able to leverage other application package formats for customization and extension would provide access to a trove of KUDO Operators without any additional work required by OperatorVendors. There are

- Operators
- Helm
- CNAB
- OperatorHub

### Goals

- Run non-KUDO defined applications as Operators

### Non-Goals

- (currently) Manage repos of non-KUDO applications (e.g. Helm Charts)
- Convert non-KUDO applications into KUDO format
- Create a plugin system to upload new templating engines. Must be present in core KUDO codebase (for now)
- Adjusting the capabilities of Parameters to support non-string values

## Proposal

### Engine Interface

The core functionality required by KUDO is the ability to turn

```go
type StepArgument struct{
  // The OperatorVersion object to be used by the Engine
  OperatorVersion OperatorVersion
  // The Plan being executed
  Plan Plan
  // The Phase being executed
  Phase Phase
  // The step that is being currently run
  Step Step
  // The parameter values from the Instance Spec
  Arguments map[string]string
}
```

into a list of Kubernetes objects. Therefore, defining an `Engine` interface for a new application spec as follows would allow for using that spec in the execution of a plan.

```go
type Engine interface{
  // Turns the input variables into a list of Kubernetes objects to apply
  // for this step
  Render(args StepArgument) ([]runtime.Objects, error)
}
```

The `PlanExecution` controller would select the proper engine based on the `engine` keyword added to the `OperatorVersion` spec.

### Engine Spec in OperatorVersion

The Operator Version Spec would have a new field `Engine` that would specify the engine that should be used to Render steps into Kubernetes objects. Initial implementation will focus on adding one additional engine to support the execution of `Helm` charts.

```go
// OperatorVersionSpec defines the desired state of OperatorVersion.
type OperatorVersionSpec struct {
	// +optional
	Operator corev1.ObjectReference operator

  ...

  // Define the engine for rendering.  Defaults to `kudo`
  Engine EngineType `json:"engine,omitempty"`
}

type EngineType string

const (
  // Use default kudo engine for rendering
  EngineTypeKUDO EngineType = "kudo"
  operator
  EngineTypeHelm EngineType = "helm"
)

```

#### KUDO Engine

The functionality and rendering of KUDO operators will not change as a result of this KEP.

#### Helm Engine

When an OperatorVersion is created from a Helm Chart, we will refer to this OperatorVersion as a **Helm OperatorVersion** to differentiate it from the underlying **Helm Chart** and an OperatorVersion that uses the KUDO Engine.

##### Installation of Helm OperatorVersions

Helm charts have a similar structure to KUDO Operators and the converstion, at installation time, of a Helm Chart into a Helm OperatorVersion should be straightfoward.

- The `Chart.yaml` will be converted:
  - Operator.name = name
  - OperatorVersion.name = name-version
  - Operator.spec.maintainers = maintainers
  - Operator.spec.url = url
  - OperatorVersion.kubernetesVersion = kubeVersion
  - OperatorVersion.kudoVersion = 0.3.0
- Files in the `templates` folder will be inserted into the `templates` field in the OperatorVersion, exactly how is done in KUDO OperatorVersions.
- There will be no parameters defined explicitly in the OperatorVersion.

By default there will be a `deploy` plan with one `deploy` phase and one `deploy` step that will render all objects in the chart.

### User Stories

- Allow running of a Helm Chart as a KUDO Operator
- Allow running of a CNAB bundle as a KUDO Operator
- Allow the running of an Operator as a KUDO Operator

### Risks and Mitigations

- It's not clear all functionality present in other application definition formats will be present in KUDO. Only support what KUDO supports currently.
- Once [KEP-0012](0012-operator-extensions.md) gets merged, we may need to adjust the interface to get rendered by the TASK, not the step so that tasks from different engines can be rendered in the same step.

## Graduation Criteria

- Be able to successfully run the following popular Helm Charts:
  - Jenkins
  - Prometheus
  - Ghost
  - Redis
  - Postgreql
  - Grafana
