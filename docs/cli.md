---
title: CLI Usage
type: docs
menu: docs
---
# CLI Usage

This document demonstrates how to use the CLI but also shows what happens in `KUDO` under the hood, which can be helpful when troubleshooting.

## Table of Contents

   * [CLI Usage](#cli-usage)
      * [Table of Contents](#table-of-contents)
      * [Kubectl KUDO Plugin](#kubectl-kudo-plugin)
         * [Requirements](#requirements)
         * [Install](#install)
         * [Commands](#commands)
         * [Flags](#flags)
         * [Environment Variables](#environment-variables)
         * [Examples](#examples)
            * [Install a Package](#install-a-package)
            * [Install a package overriding instance name and parameters](#install-a-package-overriding-instance-name-and-parameters)
            * [Get Instances](#get-instances)
            * [Get the Status of an Instance](#get-the-status-of-an-instance)
            * [Delete an Instance](#delete-an-instance)
            * [Get the History to PlanExecutions](#get-the-history-to-planexecutions)


## Setup the KUDO Kubectl Plugin

### Requirements

- `kubectl` version `1.12.0` or newer
- KUDO CRDs installed to your cluster and KUDO controller is running. See the [getting started guide](/docs/getting-started/) for instructions
- `kubectl kudo` is running outside your cluster

### Install

You can either install the CLI plugin using `brew`:

- `brew tap kudobuilder/tap`
- `brew install kudo-cli`

Or you can use compile and install the plugin from your `$GOPATH/src/github.com/kudobuilder/kudo` root folder via:

- `make cli-install`

## Commands

| Syntax                                     | Description                                                                                   |
| ------------------------------------------ | --------------------------------------------------------------------------------------------- |
| `kubectl kudo install <name> [flags]`      | Install a Operator from the official [KUDO repo](https://github.com/kudobuilder/operators). |
| `kubectl kudo get instances [flags]`       | Show all available instances.                                                                 |
| `kubectl kudo plan status [flags]`         | View all available plans.                                                                     |
| `kubectl kudo plan history <name> [flags]` | View all available plans.                                                                     |
| `kubectl kudo version`                     | Print the current KUDO package version.                                                       |

## Flags

```
Usage:
  kubectl-kudo install <name> [flags]

Flags:
      --auto-approve              Skip interactive approval when existing version found. (default "false")
  -h, --help                      help for install
      --instance string           The instance name. (default to Operator name)
      --kubeconfig string         The file path to Kubernetes configuration file. (default "$HOME/.kube/config")
      --namespace string          The namespace used for the operator installation. (default "default")
      --package-version string    A specific package version on the official GitHub repo. (default to the most recent)
  -p, --parameter stringArray     The parameter name and value separated by '='
```

## Examples

### Install a Package

There are two options how to install a package. For development you want to install packages from your local filesystem.

```bash
kubectl kudo install pkg/kudoctl/util/repo/testdata/zk
```

For normal operations it is recommended to use the official packages provided through the [kudobuilder/operators](https://github.com/kudobuilder/operators) repository.
To install official kafka package you have to do the following:

```bash
kubectl kudo install kafka
```

Both of these options will install new instance of that operator into your cluster. By default the instance name is generated.

#### Install a package overriding instance name and parameters

Use `--instance` and `--parameter`/`-p` for setting an Instance name and Parameters, respectively:

```bash
$ kubectl kudo install kafka --instance=my-kafka-name --parameter ZOOKEEPER_URI=zk-zk-0.zk-hs:2181,zk-zk-1.zk-hs:2181,zk-zk-2.zk-hs:2181 --parameter ZOOKEEPER_PATH=/small -p BROKERS_COUNTER=3
operator.kudo.dev/kafka unchanged
operatorversion.kudo.dev/kafka unchanged
No instance named 'my-kafka-name' tied to this "kafka" version has been found. Do you want to create one? (Yes/no)
instance.kudo.dev/v1alpha1/my-kafka-name created
$ kubectl get instances
NAME            AGE
my-kafka-name   6s
```


### Get Instances

We can use the `get` command to get a list of all current instances:

`kubectl kudo get instances --namespace=<default> --kubeconfig=<$HOME/.kube/config>`

Example:

```bash
$ kubectl kudo get instances
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
$ kubectl kudo get instances
  NAME      CREATED AT
  small     4d
  up        3d
  zk        4d
```

### Get the Status of an Instance

Now that we have a list of available instances, we can get the current status of all plans for one of them:

`kubectl kudo plan status --instance=<instanceName> --kubeconfig=<$HOME/.kube/config>`

*Note: The `--instance` flag is **mandatory**.*

```bash
$ kubectl kudo plan status --instance=up
  Plan(s) for "up" in namespace "default":
  .
  └── up (Operator-Version: "upgrade-v1" Active-Plan: "up-deploy-493146000")
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

* `up` is the instance we specified.
* `default` is the namespace we are in.
* `upgrade-v1` is the instance's **Operator-Version**.
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

For comparison, the according `kubectl` commands to retrieve the above information are:

* `kubectl get instances` (to get the matching `OperatorVersion`)
* `kubectl describe operatorversion upgrade-v1` (to get the current `PlanExecution`)
* `kubectl describe planexecution up-deploy-493146000` (to get the status of the `Active-Plan`)

Here, the overview of all available plans can be found in `Spec.Plans` of the matching `OperatorVersion`:

```bash
$ kubectl describe operatorversion upgrade-v1
Name:         upgrade-v1
Namespace:    default
Labels:       controller-tools.k8s.io=1.0
Annotations:  kubectl.kubernetes.io/last-applied-configuration={"apiVersion":"kudo.dev/v1alpha1","kind":"OperatorVersion","metadata":{"annotations":{},"labels":{"controller-tools.k8s.io":"1.0"},"name":"upgra...
API Version:  kudo.dev/v1alpha1
Kind:         OperatorVersion
Metadata:
  Cluster Name:
  Creation Timestamp:  2018-12-14T19:26:44Z
  Generation:          1
  Resource Version:    63769
  Self Link:           /apis/kudo.dev/v1alpha1/namespaces/default/operatorversions/upgrade-v1
  UID:                 30fe6209-ffd6-11e8-abd5-080027d506c7
Spec:
  Connection String:
  Operator:
    Kind:  Operator
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

The status of the currently applied plan can then be found when looking at the particular `PlanExecution`:

```bash
$ kubectl describe planexecution up-deploy-493146000
  Name:         up-deploy-493146000
  Namespace:    default
  Labels:       operator-version=upgrade-v1
                instance=up
  Annotations:  <none>
  API Version:  kudo.dev/v1alpha1
  Kind:         PlanExecution
  Metadata:
    Cluster Name:
    Creation Timestamp:  2018-12-14T19:26:44Z
    Generation:          1
    Owner References:
      API Version:           kudo.dev/v1alpha1
      Block Owner Deletion:  true
      Controller:            true
      Kind:                  Instance
      Name:                  up
      UID:                   3101bbe5-ffd6-11e8-abd5-080027d506c7
    Resource Version:        63815
    Self Link:               /apis/kudo.dev/v1alpha1/namespaces/default/planexecutions/up-deploy-493146000
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

Finally, the status information for the `Active-Plan` is nested in this part:

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

Apparently, KUDO's tree view makes this information easier to understand and prevents you from putting together the bits and pieces of various commands.

### Delete an Instance

An instance can be deleted (uninstalled from the cluster) using `kubectl delete instances <instanceName>`. The deletion of an instance triggers the removal of all the objects owned by it.

### Get the History to PlanExecutions

This is helpful if you want to find out which plan ran on your instance to a particular `OperatorVersion`.
Run this command to retrieve all plans that ran for the instance `up` and its OperatorVersion `upgrade-v1`:

```bash
$ kubectl kudo plan history upgrade-v1 --instance=up
  History of plan-executions for "up" in namespace "default" to operator-version "upgrade-v1":
  .
  └── up-deploy-493146000 (created 4h56m12s ago)
```

Run this command to retrieve the history of all plans applied to an instance:

```bash
$ kubectl kudo plan history --instance=up
  History of all plan-executions for "up" in namespace "default":
  .
  └── up-deploy-493146000 (created 4h52m34s ago)
```

This includes the previous history but also all OperatorVersions that have been applied to the selected instance.
