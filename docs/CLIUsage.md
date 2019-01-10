# CLI Usage

This document demonstrates how to use the CLI but also shows what happens in `Maestro` under the hood, which can be helpful when troubleshooting.

The usage is intended to make your life easier when working with `Maestro`. A work flow would look lik this:

1) You get a list of all available Instances deployed by `Maestro`
2) You get the status the particular Instance of interest
3) You see a history to a specific Instance

# List Instances

In order to inspect instances deployed by `Maestro` we need to get an overview of all instances running.
Therefor we use the `list` command which has subcommands to show all available instances:

`maestroctl list instances --namespace=<default> --kubeconfig=<$HOME/.kube/config>`

Example:

```bash
$ maestroctl list instances
  List of current instances in namespace "default":
  .
  ├── small
  ├── up
  └── zk
```

This maps to the `kubectl` command: 

`kubectl get instances`

Example:

```bash
$ kubectl get instances
  NAME      CREATED AT
  small     4d
  up        3d
  zk        4d
```

# Get the Status of an Instance

(Or how to look up your instance status)

Now that we know the available instances we can get the current status of all plans to an particular instance. The command for this is:

`maestroctl plan status --instance=<instanceName> --kubeconfig=<$HOME/.kube/config>`

*Note: The `--instance` flag is mandatory and **not optional**.*

```bash
$ maestroctl plan status --instance=up
  Plan(s) for "up" in namespace "default":
  .
  └── up (Framework-Version: "upgrade-v1" Active-Plan: "up-deploy-493146000")
      ├── Plan deploy (serial strategy) [COMPLETE]
      │   └── Phase par (serial strategy) [COMPLETE]
      │       └── Step run-step (COMPLETE)
      ├── Plan update (serial strategy) [NOT ACTIVE]
      │   └── Phase par (serial strategy) [NOT ACTIVE]
      │       └── Step par (serial strategy) [NOT ACTIVE]
      │           └── run-step [NOT ACTIVE]
      └── Plan upgrade (serial strategy) [NOT ACTIVE]
          └── Phase par (serial strategy) [NOT ACTIVE]
              └── Step par (serial strategy) [NOT ACTIVE]
                  └── run-step [NOT ACTIVE]
```

In this tree chart we see all important information in one screen:

* `up` is the current instance we are looking at.
* `default` is the current namespace we are in.
* `upgrade-v1` is the current **Framework-Version**.
* `up-deploy-493146000` is the current **Active-Plan**.
    + `par` is a serial phase within the `deploy` plan that has been `COMPLETE`
    + `deploy` is a `serial` plan that has been `COMPLETE`.
    + `run-step` is a `serial` step that has been `COMPLETE`.
* `update` is another `serial` plan that is currently `NOT ACTIVE`.
    + `par` is a serial phase within the `update` plan that has been `NOT ACTIVE`
    + `par` is a `serial` collection of steps that has been `NOT ACTIVE`.
    + `run-step` is a `serial` step within the `par` step collection that has been `NOT ACTIVE`.
* `upgrade` is another `serial` plan that is currently `NOT ACTIVE`.
    + `par` is a serial phase within the `upgrade` plan that has been `NOT ACTIVE`
    + `par` is a `serial` collection of steps that has been `NOT ACTIVE`.
    + `run-step` is a `serial` step within the `par` step collection that has been `NOT ACTIVE`.
    
This would translate to the following manual `kubectl` commands:

* `kubectl get instances` (to get the matching `FrameworkVersion`)
* `kubectl describe frameworkversion upgrade-v1` (to get the current `PlanExecution`)
* `kubectl describe planexecution up-deploy-493146000` (to get the status of the `Active-Plan`)

The overall overview of all available plans can be found in `Spec.Plans` of the matching `FrameworkVersion`.

```bash
$ kubectl describe frameworkversion upgrade-v1
Name:         upgrade-v1
Namespace:    default
Labels:       controller-tools.k8s.io=1.0
Annotations:  kubectl.kubernetes.io/last-applied-configuration={"apiVersion":"maestro.k8s.io/v1alpha1","kind":"FrameworkVersion","metadata":{"annotations":{},"labels":{"controller-tools.k8s.io":"1.0"},"name":"upgra...
API Version:  maestro.k8s.io/v1alpha1
Kind:         FrameworkVersion
Metadata:
  Cluster Name:        
  Creation Timestamp:  2018-12-14T19:26:44Z
  Generation:          1
  Resource Version:    63769
  Self Link:           /apis/maestro.k8s.io/v1alpha1/namespaces/default/frameworkversions/upgrade-v1
  UID:                 30fe6209-ffd6-11e8-abd5-080027d506c7
Spec:
  Connection String:  
  Framework:
    Kind:  Framework
    Name:  upgrade
  Parameters:
    Default:       15
    Description:   how long to have the container sleep for before returning
    Display Name:  Sleep Time
    Name:          SLEEP
    Required:      false
  Plans:
    Deploy:
      Phases:
        Name:  par
        Steps:
          Name:  run-step
          Tasks:
            run
        Strategy:  serial
      Strategy:    serial
    Update:
      Phases:
        Name:  par
        Steps:
          Name:  run-step
          Tasks:
            run
        Strategy:  serial
      Strategy:    serial
    Upgrade:
      Phases:
        Name:  par
        Steps:
          Name:  run-step
          Tasks:
            run
        Strategy:  serial
      Strategy:    serial
  Tasks:
    Run:
      Resources:
        job.yaml
  Templates:
    Job . Yaml:  apiVersion: batch/v1
kind: Job
metadata:
  namespace: default
  name: {{PLAN_NAME}}-job
spec:
  template:
    metadata:
      name: {{PLAN_NAME}}-job
    spec:
      restartPolicy: OnFailure
      containers:
      - name: bb
        image: busybox:latest
        imagePullPolicy: IfNotPresent
        command:
        - /bin/sh
        - -c
        - "echo {{PLAN_NAME}} for v1 && echo Going to sleep for {{SLEEP}} seconds && sleep {{SLEEP}}"

  Version:  1.0.0
Events:     <none>
```

When looking at the particular `PlanExecution` we are able to see the status of the current applied plan.

```bash
$ kubectl describe planexecution up-deploy-493146000
  Name:         up-deploy-493146000
  Namespace:    default
  Labels:       framework-version=upgrade-v1
                instance=up
  Annotations:  <none>
  API Version:  maestro.k8s.io/v1alpha1
  Kind:         PlanExecution
  Metadata:
    Cluster Name:        
    Creation Timestamp:  2018-12-14T19:26:44Z
    Generation:          1
    Owner References:
      API Version:           maestro.k8s.io/v1alpha1
      Block Owner Deletion:  true
      Controller:            true
      Kind:                  Instance
      Name:                  up
      UID:                   3101bbe5-ffd6-11e8-abd5-080027d506c7
    Resource Version:        63815
    Self Link:               /apis/maestro.k8s.io/v1alpha1/namespaces/default/planexecutions/up-deploy-493146000
    UID:                     31037dd0-ffd6-11e8-abd5-080027d506c7
  Spec:
    Instance:
      Kind:       Instance
      Name:       up
      Namespace:  default
    Plan Name:    deploy
  Status:
    Name:  deploy
    Phases:
      Name:   par
      State:  COMPLETE
      Steps:
        Name:    run-step
        State:   COMPLETE
      Strategy:  serial
    State:       COMPLETE
    Strategy:    serial
  Events:        <none>
```

The status information for the `Active-Plan` is nested in this part:

```bash
  Status:
    Name:  deploy
    Phases:
      Name:   par
      State:  COMPLETE
      Steps:
        Name:    run-step
        State:   COMPLETE
      Strategy:  serial
    State:       COMPLETE
    Strategy:    serial
```

Our tree view makes this information more readable to the user and creates a better user experience through one command rather than picking from multiple responses the bits and pieces.

# Get the History to PlanExecutions

This is helpful if you want to find out which plan ran on your instance to a particular `FrameworkVersion`.
Let's say you want to know all plans that ran for the instance `up` and its FrameworkVersion `upgrade-v1`:

```bash
$ maestroctl plan history upgrade-v1 --instance=up
  History of plan-executions for "up" in namespace "default" to framework-version "upgrade-v1":
  .
  └── up-deploy-493146000 (created 4h56m12s ago)
```

If you want to have a broader history to all plans applied to an instance:

```bash
$ maestroctl plan history --instance=up
  History of all plan-executions for "up" in namespace "default":
  .
  └── up-deploy-493146000 (created 4h52m34s ago)
```

This includes the previous history but also all FrameworkVersions that have been applied to the selected instance.