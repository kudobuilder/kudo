---
kep-number: 27
short-desc: Detailed control what parameter changes trigger pod restarts
title: Pod restart control
authors:
  - @aneumann82
owners:
  - @aneumann82
editor:
creation-date: 2020-03-31
last-updated: 2020-04-15
status: provisional
see-also:
replaces:
superseded-by:
---

## Summary

This KEP describes a feature to allow detailed control when templated pods in a stateful set are restarted when parameters
that are not used in the template change.

## Motivation

KUDO currently has a feature that automatically restarts Pods of Deployments or StatefulSets if any KUDO-Parameter is changed. This
is often required as a Parameter can change a value in a ConfigMap that is used by the Pods. If the Pods are not restarted
the change is propagated to the Pods, but as most applications are not aware of changing config files they require a 
restart. This was originally solved by using [stakater/Reloader](https://github.com/stakater/Reloader) but this had 
[issues](https://github.com/kudobuilder/kudo/issues/1025) which lead to the self developed solution, which is at the 
moment applied to [all templates](https://github.com/kudobuilder/kudo/issues/1036)

This behavior can currently not be controlled and is always active. The issue is that not all Parameter changes require
a rolling restart of Pods, and sometimes the restart is unwanted and can negatively affect the state of the application
the operator is controlling.

One example would be the `replica` count of a StatefulSet: An increase here should only start new Pods and not restart
all existing ones. Another example would be an application that regularly re-reads the config files, in this case the 
pod doesn't need to be restarted if a config map changes - the application would pick up changed values automatically.

A new flag introduced by this KEP will allow an operator developer to control when a changed parameter will trigger a 
restart of pods from a deployment or stateful set.

### Goals

Make it possible for an operator developer to specify when a parameter will restart all pods from a stateful set or a deployment.

### Non-Goals

- Preventing Pod restart where plain k8s would do it: If the spec template for a stateful set is changed, Kubernetes will still
restart all pods automatically - there is no way to prevent that.
- Calculation of used config maps, secrets or other resources used by a pod: The automatic pod restart is currently 
used to force pods to re-read config maps on a parameter update. When a user disables the pod restart for 
a parameter, and this parameter updates a config map or another dependency of the pod, the change might not be applied
in the pod until the next restart.
- Restarting a stuck Pod

## Proposal

Allow the operator developer to define groups of parameters that can then be used to trigger a restart of pods from
a deployment or stateful set.

Add a list of `groups` that each parameter is part of:

```yaml
  - name: CONFIG_VALUE
    description: "A parameter that changes a config map which is used by a stateful set."
    default: "3"
    groups:
      - MAIN_GROUP
  - name: SOME_OTHER_PARAMETER
    description: "A parameter that changes something that is used by two stateful sets."
    default: "3"
    groups:
      - MAIN_GROUP
      - ADDITIONAL_GROUP
```

KUDO then calculates the hash of all parameters in a group. This hash can then be used
instead of the the `last-plan-execution-uid` in a deployment or stateful set to trigger the reload of the pods:

```yaml
spec:
  template:
    metadata:
      annotations:
        config-values-hash: {{ .ParamGroups.MAIN_GROUP.hash }}
```

When a parameter in the group changes, the hash will change, which in turn would then trigger the restart of the
pods.

This proposal would remove the `last-plan-execution-uid` and automatic pod restarts. Configuration for
pod restarts would be fully in the responsibility of the operator developer.  

It may be a good idea to define a `default` or `ALL` group that is always the set of all defined parameters.

### User Stories

- [#1424](https://github.com/kudobuilder/kudo/issues/1424) Fine control over pod restarts
- [#1036](https://github.com/kudobuilder/kudo/issues/1036) Restarting all pods when scaling up

The main use case for this extension are big stateful sets that are sensitive to pod restarts, for example Cassandra. 

Especially while changing the size of a stateful set, the pod definition itself is not modified at all and a full
restart of all pods will have a negative impact.

### Implementation Details/Notes/Constraints

The forced pod restart is currently implemented by setting a UID in the attributes of the `podTemplate`.

Generally, the enhancer should not modify any template specs, but only the top-level resource.

At the moment, the enhancer adds labels and attributes to templates:
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
The labels are very static no change is required here. Labels can be used as selectors, and it still will be
beneficial to have the heritage, operator and instance labels on templated pods.


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
The annotations are volatile and will/may change with every plan execution. The nested changes to template and jobTemplate
are historically made because the behavior was modeled after Kustomize.

This KEP proposes to change this: 
- Attributes will only be applied to the top-level resource, not nested resources ( `spec/template/...`, `spec/jobTemplate/...`)
- The PlanUIDAnnotation will be completely removed.

The values of the attributes can still be discovered by following the ownerReference of created resources.

### Risks and Mitigations

The proposed change removes the automatic pod restart. It will be a breaking change, operator developers will need
to adjust the operator to define a group for all used parameters and add their own attribute in the pod spec template
that triggers the restart. This needs to be documented, and old operators will behave differently with the new KUDO
version.

## Implementation History

- 2020-03-31 - Initial draft. (@aneumann)
- 2020-04-14 - Add second Proposal
- 2020-04-15 - Cleanup, moved first proposal to alternatives section

## Alternatives

### Full dependency graph for used resources and parameters

A better alternative would be to have a full dependency graph of all resources to parameters. This would need to include
ConfigMaps, Secrets, potentially other pods, etc. With this dependency graph, KUDO could determine when a pod restart
is necessary and when not, i.e. changing the replica value of a stateful set would not trigger a pod restart, changing
a value in a ConfigMap that is used by the pod template would trigger a restart.
An operator would still want to configure if and when pod restarts may be required, as a pod can be aware of changing
ConfigMaps. 

Calculating the dependency graph would be a very complex undertaking, and may not even be possible in all cases. The
implementation effort would be very high for only medium additional benefits

### ForcePodRestart attribute for parameters

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

This solution would be very easy to implement, but may be hard to explain and requires intimate knowledge of KUDO internals
for operator developers. It also would not be possible to have different handling for multiple stateful sets or 
deployments.