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
status: implementable
see-also:
replaces:
superseded-by:
---

## Summary

This KEP describes a feature to allow detailed control when templated pods in a stateful set and deployments are restarted when parameters that are not used in the template change.

## Motivation

KUDO currently has a feature that automatically restarts Pods of Deployments or StatefulSets if any KUDO-Parameter is changed. This is often required as a Parameter can change a value in a ConfigMap that is used by the Pods. If the Pods are not restarted the change is propagated to the Pods, but as most applications are not aware of changing config files they require a restart. 
This was originally solved by using [stakater/Reloader](https://github.com/stakater/Reloader) but this had [issues](https://github.com/kudobuilder/kudo/issues/1025) which lead to the self developed solution, which is at the moment applied to [all templates](https://github.com/kudobuilder/kudo/issues/1036)

This behavior currently can not be controlled and is always active. The issue is that not all Parameter changes require a rolling restart of Pods, and sometimes the restart is unwanted and can negatively affect the state of the application the operator is controlling.

One example would be the `replica` count of a StatefulSet: An increase here should only start new Pods and not restart all existing ones. Another example would be an application that regularly re-reads the config files, in this case the pod doesn't need to be restarted if a config map changes - the application would pick up changed values automatically.

A new flag introduced by this KEP will allow an operator developer to control when a changed parameter will trigger a restart of pods from a deployment or stateful set.

### Goals

Make it possible for an operator developer to specify when a parameter will restart all pods from a stateful set or a deployment.

### Non-Goals

- Preventing Pod restart where plain k8s would do it: If the spec template for a stateful set is changed, Kubernetes will still restart all pods automatically - there is no way to prevent that.
- Calculation of used config maps, secrets or other resources used by a pod: The automatic pod restart is currently used to force pods to re-read config maps on a parameter update. When a user disables the pod restart for a parameter, and this parameter updates a config map or another dependency of the pod, the change might not be applied in the pod until the next restart.
- Restarting a stuck Pod (A step that cannot be executed, a pod that cannot be deployed because of various reasons, a pod that crash loops, etc.)

## Proposal - Resource Dependency Graph

KUDO will analyze the operator and build a dependency graph of used resources:

- Each deployment, stateful set and batch job will be analysed which results in list of required resources (ConfigMaps and Secrets).
- When a resource is deployed, KUDO calculates a hash from all dependent resources and adds it as an annotation to the pod template. This will replace the current `last-plan-execution-uid`.

The resources required by a top-level resource may not necessarily be deployed in the same step as the resource itself. This can lead to an update of a required resource in a step that does not deploy the top-level resource and vice versa. To correctly calculate the hash of the required resources this needs to be done different ways:
- For resources that are deployed in the current step: Calculate hashes of the resources from the rendered templates parameters
- For resources that are *not* deployed in the current step: Fetch the resources from the api-server and get the `kudo.LastAppliedConfigAnnotation` annotation to calculate the hash.

This makes sure that the hash is correctly calculated even if a required resource was changed by a different plan in the meantime.

There are use cases where a changed resource does not require a pod restart. To allow a resource to be *not* included in the hash calculation, the user can set a special annotation on a config map or secret.

### User Stories

- [#1424](https://github.com/kudobuilder/kudo/issues/1424) Fine control over pod restarts
- [#1036](https://github.com/kudobuilder/kudo/issues/1036) Restarting all pods when scaling up

The main use case for this extension are big stateful sets that are sensitive to pod restarts, for example Cassandra. 

Especially while changing the size of a stateful set, the pod definition itself is not modified at all and a full restart of all pods will have a negative impact.

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
The annotations are volatile and will/may change with every plan execution. The nested changes to template and jobTemplate are historically made because the behavior was modeled after Kustomize.

This KEP proposes to change this: 
- Annotations will only be applied to the top-level resource, not nested resources ( `spec/template/...`, `spec/jobTemplate/...`). This means it will not be possible to directly discover the plan, phase or step by which a pod was deployed. It will require to follow the ownerReference to find these values in the parent resource (StatefulSet, Deployment, BatchJob, etc.) 
- The PlanUIDAnnotation will be removed and replaced by a calculated hash from the used resources.

### Risks and Mitigations

The proposed change removes the automatic pod restart. It will be a breaking change, operator developers will need to adjust the operator to define a group for all used parameters and add their own attribute in the pod spec template that triggers the restart. This needs to be documented, and old operators will behave differently with the new KUDO version.

## Implementation History

- 2020-03-31 - Initial draft. (@aneumann)
- 2020-04-14 - Add second Proposal
- 2020-04-15 - Cleanup, moved first proposal to alternatives section
- 2020-04-24 - Discarded parameter group proposal

## Alternatives

### ForcePodRestart attribute for parameters

Add an additional attribute `forcePodRestart` to parameter specifications in `params.yaml`:

```yaml
  - name: NODE_COUNT
    description: "Number of Cassandra nodes."
    default: "3"
    forcePodRestart: "false"
```

The default value for this parameter would be `true` to keep backwards compatibility. The general behavior should stay the way it is right now, that on a change of a parameter the pods will be automatically restarted.

If multiple parameters are changed in one update, the `forcePodRestart` flags of all attributes are `OR`ed: If at least one parameter has the `forcePodRestart` set to `true`, the pods will execute a rolling restart.

This solution would be very easy to implement, but may be hard to explain and requires intimate knowledge of KUDO internals for operator developers. It also would not be possible to have different handling for multiple stateful sets or deployments.

## Parameter Groups

Allow the operator developer to define groups of parameters that can then be used to trigger a restart of pods from a deployment or stateful set.

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

KUDO then calculates the hash of all parameters in a group. This hash can then be used instead of the the `last-plan-execution-uid` in a deployment or stateful set to trigger the reload of the pods:

```yaml
spec:
  template:
    metadata:
      annotations:
        config-values-hash: {{ .ParamGroupHashses.MAIN_GROUP }}
```

When a parameter in the group changes, the hash will change, which in turn would then trigger the restart of the pods.

This proposal would remove the `last-plan-execution-uid` and automatic pod restarts. Configuration for pod restarts would be fully in the responsibility of the operator developer.  

There will be a default `ALL` group that all parameters belong to. For an operator developer to restore the current behavior, the would only need to add the following annotation to the spec template:

```yaml
spec:
  template:
    metadata:
      annotations:
        config-values-hash: {{ .ParamGroupHashses.ALL }}
```

This proposal was discarded as param groups would put a big burden on operator developers to keep track of groups and would introduce a whole new paradigm for a limited outcome.