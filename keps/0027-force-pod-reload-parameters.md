---
kep-number: 27
short-desc: A parameter flag to disable forced pod reload
title: Forced Pod reload attribute
authors:
  - @aneumann82
owners:
  - @aneumann82
editor:
creation-date: 2020-03-31
last-updated: 2020-03-31
status: provisional
see-also:
replaces:
superseded-by:
---

## Summary

This KEP describes addition of a flag for parameters that controls the forced pod reloading

## Motivation

The new flag will allow an operator developer to control that a specific attribute does not automatically restart
all pods of a deployment or stateful set if this parameter is changed.

### Goals

Make it possible for an operator developer to specify that a parameter will *not* automatically restart all pods from
a stateful set or a deployment.

### Non-Goals

- Detailed control when Pods are restarted: If the spec template for a stateful set is changed, Kubernetes will still
reload all pods automatically.
- Calculation of used config maps, secrets or other resources used by a pod: The automatic pod restart is currently 
used to force pods to re-read config maps on a parameter update. When a user explicitly disables the pod restart for 
a parameter, and this parameter updates a config map or another dependency of the pod, the change might not be applied
in the pod until the next restart.

## Proposal

Add an additional attribute `forcePodRestart` to parameter specifications in `params.yaml`:

```yaml
  - name: NODE_COUNT
    description: "Number of Cassandra nodes."
    default: "3"
    forcePodRestart: "false"
```

The default value for this parameter would be `true` to keep backwards compatibility. The general behavior should stay
the way it is right now, that on a change of a parameter the pods will be automatically restarted.

If multiple parameters are changed in one update, the `forcePodRestart` flags of all attributes are `OR`ed: If at least one
parameter has the `forcePodRestart` set to `true`, the pods will execute a rolling restart.

### User Stories

- [#1424](https://github.com/kudobuilder/kudo/issues/1424) Fine control over pod restarts

The main use case for this extension are big stateful sets that are sensitive to pod restarts, for example Cassandra. 

Especially while changing the size of a stateful set, the pod definition itself is not modified at all and a full
restart of all pods will have a negative impact.

### Implementation Details/Notes/Constraints

The forced pod restart is currently implemented by setting a UID in the attributes of the `podTemplate`.

If the added parameter `forcePodRestart` is set to false, this UID should not be set, therefore keeping the podTemplate
unchanged. This would prevent a rolling restart of all pods.

Generally, the enhancer should not modify any template specs, but only the top-level resource if the flag is disabled.

One option to consider is changing the behavior of the enhancer in general: At the moment, the enhancer adds labels and
attributes to templates:
```
	fieldsToAdd := map[string]string{
		kudo.HeritageLabel: "kudo",
		kudo.OperatorLabel: metadata.OperatorName,
		kudo.InstanceLabel: metadata.InstanceName,
	}
	labelPaths := [][]string{
		{"metadata", "labels"},
		{"spec", "template", "metadata", "labels"},
		{"spec", "volumeClaimTemplates[]", "metadata", "labels"},
		{"spec", "jobTemplate", "metadata", "labels"},
		{"spec", "jobTemplate", "spec", "template", "metadata", "labels"},
	}

```
The labels are very static no change is required here.


```
	fieldsToAdd := map[string]string{
		kudo.PlanAnnotation:            metadata.PlanName,
		kudo.PhaseAnnotation:           metadata.PhaseName,
		kudo.StepAnnotation:            metadata.StepName,
		kudo.OperatorVersionAnnotation: metadata.OperatorVersion,
		kudo.PlanUIDAnnotation:         string(metadata.PlanUID),
	}
	annotationPaths := [][]string{
		{"metadata", "annotations"},
		{"spec", "template", "metadata", "annotations"},
		{"spec", "jobTemplate", "metadata", "annotations"},
		{"spec", "jobTemplate", "spec", "template", "metadata", "annotations"},
	}
```
The annotations are volatile and will/may with every plan execution. The nested changes to template and jobTemplate
are historically made because the behavior was modeled after Kustomize.

This KEP proposes to change this: Attributes will only be applied to the top-level resource, not nested resources.

The PlanUIDAnnotation *will* be set on nested resources and templates, depending on the new `forcePodReload` attribute.

The values of the attributes can still be discovered by following the ownerReference of created resources.

### Risks and Mitigations

As we would skip the update of the template specs of pods, we would not write the latest plan, phase and step into
the attributes of the pods. This could lead to incorrect data in the attributes, as the pod might still get restarted
if the modified parameter is used in the template specs - it would be in the responsibility of the operator developer
to prevent this scenario. 

This new attribute has a very special meaning: It basically just prevents the update of attributes on template specs. It
might be a too complex concept. This should be mitigated by careful naming of the new attribute and documentation.

## Implementation History

- 2020-03-31 - Initial draft. (@aneumann)

## Alternatives

A better alternative would be to have a full dependency graph of all resources to parameters. This would need to include
ConfigMaps, Secrets, potentially other pods, etc. Calculating the dependency graph would be a very complex undertaking,
a simple configuration on a parameter seems to be a more reasonable approach.
