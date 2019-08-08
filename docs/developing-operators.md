---
title: Developing operators
type: docs
menu: docs
---

This guide will provide introduction to creating KUDO operators, you will learn about the structure of the package and the template language to use.

## Getting started
In this section we’ll start by developing your first operator and we’ll follow up with in-depth explanation of the underlying concepts.

The overall structure of a package looks following:
```shell
.
├── operator.yaml
├── params.yaml
└── templates
    ├── deployment.yaml
    └── ...
```

The `operator.yaml` is the main yaml file defining both operator metadata as the whole lifecycle of the operator. `params.yaml` defines parameters of the operator. During installation, these parameters can be overriden allowing customization. `templates` folder contain all templated kubernetes objects that will be applied to your cluster after installation based on the workflow defined in `operator.yaml`.

### Your first KUDO operator
First let’s create `operator.yaml` and place it in a `first-operator` folder.

```yaml
name: "first-operator"
version: "0.1.0"
maintainers:
- Your name <your@email.com>
url: https://kudo.dev
tasks:
  nginx:
    resources:
      - deployment.yaml
plans:
  deploy:
    strategy: serial
    phases:
      - name: main
        strategy: parallel
        steps:
          - name: everything
            tasks:
              - nginx
```

As you can see we have operator with just one plan - deploy, with one phase and one step which is the minimal setup. Deploy plan is automatically triggered when you install instance of this operator into cluster.

You can see that the task `nginx` references the resource `deployment.yaml`. KUDO automatically expects this file to exist inside `templates` folder. So let’s create `templates/deployment.yaml`.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  selector:
    matchLabels:
      app: nginx
  replicas: {{ .Params.Replicas }} # tells deployment to run X pods matching the template
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: nginx:1.7.9
          ports:
            - containerPort: 80
```

This looks like pretty normal kubernetes yaml defining deployment and you’re right. But you can already see the KUDO templating language in action on the line referencing `{{ .Params.Replicas }}`. This will get substituted during installation by merging what is in `params.yaml` and overrides defined before install. So let’s define the last missing piece, `params.yaml`.

```yaml
replicas:
  description: Number of replicas that should be run as part of the deployment
  default: 2
```

Now your first operator is ready and you can install it to cluster. You can do it by invoking `kubectl kudo install ./first-operator` where `./first-operator` is relative path to the folder containing your operator. To do this you need KUDO CLI installed - [follow the instructions here](https://kudo.dev/docs/cli/).

For simplicity if you want to install what we created here without actually replicating it on your filesystem your can just clone KUDO repository and then run `kubectl kudo install ./config/samples/first-operator`.

## Operator.yaml file

This is the main piece of every operator. It consists of two main parts. First one defines metadata about your operator.

```yaml
name: operator
description: my first super awesome operator
version: "5.7"
kudoVersion: ">= 0.2.0"
kubernetesVersion: ">= 1.14"
maintainers:
  - Bob <bob@example.com>
  - Alice <alice@example.com>
url: https://github.com/myoperator/myoperator
```

Most of these are provided as a form of documentation. `kudoVersion` and `kubernetesVersion` use semver constraints to define minimal or maximal version of kubernetes or kudo that this operator supports. Under the hood, we use [this library](https://github.com/Masterminds/semver) to evaluate the constraints.

### Tasks section

Another part of `operator.yaml` is the tasks section. Tasks are the smallest pieces of work that get executed together. You usually group Kubernetes manifests that should be applied at once into one task. An example can be a deploy task that will result in `config.yaml` and `pod.yaml` being applied to your cluster.

```yaml
tasks:
  deploy-task:
    resources:
      - config.yaml
      - pod.yaml
```

### Plans section

Plans orchestrate tasks through `phases` and `steps`.

Each Plan is a tree with a fixed three-level hierarchy of the plan itself, its phases (named collection of steps), and then steps within those phases. This three-level hierarchy can look as follows:

```text
Plan foo
├─ Phase bar
│  ├─ Step qux
│  └─ Step quux
└─ Phase baz
   ├─ Step quuz
   ├─ Step corge
   └─ Step grault
```


Plans consists of one or more `phases`. `Phases` consists of one or more `steps`. `Steps` contain one or more `tasks` (those are defined in the section we talked about in the last paragraph). Both phases and also steps can be configured with an execution `strategy`, either `serial` or `parallel`.

The sample has a `deploy` plan with a `deploy-phase` and a `deploy-step`. From the `deploy-step` the `deploy-task` is referenced. This task gets executed when an instance is created using the operator.

At the same time, `deploy` plan is the most important plan within your operator because that is the default plan that every operator has to have and also the plan that gets executed when you install an instance of your operator into the cluster. Another important plan that you might consider having is `update` (run when instance metadata is updated) or `upgrade` (run when instance is upgraded from one version of the operator to another). If you don't provide `update` and/or `upgrade` plans for your operator, the fallback is always `deploy`.
 
```yaml
plans:
  deploy:
    strategy: serial
    phases:
      - name: deploy-phase
        strategy: parallel
        steps:
          - name: deploy-step
            tasks:
              - deploy-task
```

Plans allow operators to see what the operator is currently doing, and to visualize the broader operation such as for a config rollout. The fixed structure of that information meanwhile makes it straightforward to build UIs and tooling on top.

## Params file

`params.yaml` defines all parameters that can customize your operator installation. You have to define name of the parameter and optionally a default value. If not specified otherwise, all parameters in this list are treated as required parameters and a parameter not having a default value must have value provided during installation otherwise the installation will fail.

More detailed example of `params.yaml` may look as following:

```yaml
backupFile:
  description: "The name of the backup file"
  default: backup.sql
password:
  default: password
  description: "Password for the mysql instance"
  trigger: deploy
notrequiredparam:
  description: "This parameter is not required"
  required: false
nodefaultparam:
  description: "This parameter is required but does not have default provided"
``` 

`backupFile` parameters provides a default value so if user does not need to override it, no special action is required. A little bit different case is `notrequiredparam` which explicitly states as not being required so even though without default value, not providing value for this parameter won't fail installation. Third case is `nodefaultparam` that is required but does not provide a default value. For such parameters, user is expected to provide value for that parameter via `kubectl kudo install youroperator -p nodefaultparam=value`.

`password` parameter documents one more feature of `params.yaml` and that is trigger. Trigger is an optional field and it has to refer to an existing plan name in `operator.yaml`. When you update this parameter after instance is installed, this is the plan that gets triggered as a result of that (the plan that is going to apply changes in the parameter to your kubernetes objects). If no trigger is specified, `update` plan will be run. If no update plan exists for this operator, `deploy` plan is run.

## Templates

Everything that is placed into the templates folder is treated as template and passed on to the KUDO controller for rendering. KUDO uses [Sprig template library](https://godoc.org/github.com/Masterminds/sprig) to render your templates on server side during installation. Thanks to Sprig you can use tens of different functions inside your templates. Some of them are inherited from [go templates](https://godoc.org/text/template), some of them are defined by [Sprig](https://godoc.org/github.com/Masterminds/sprig) itself. See their documentation for full reference.

### Variables provided by KUDO

- `{{ .OperatorName }}` - name of the operator the template belongs to
- `{{ .Name }}` - name of the instance that kubernetes object created from this template will belong to
- `{{ .Namespace }}` - namespace in which the instance lives
- `{{ .Params }}` - object containing list of parameters you defined in `params.yaml` with overrides provided during installation

An more complex example using some of the built in variables could look as the following `templates/service.yaml`

```yaml
apiVersion: v1
kind: Service
metadata:
  name: svc
  {{ if eq .Params.METRICS_ENABLED "true" }}
  labels:
    "kudo.dev/servicemonitor": "true"
  {{ end }}
spec:
  ports:
    - port: {{ .Params.BROKER_PORT }}
      name: server
    - port: {{ .Params.CLIENT_PORT }}
      name: client
    {{ if eq .Params.METRICS_ENABLED "true" }}
    - port: {{ .Params.METRICS_PORT }}
      name: metrics
    {{ end }}
  clusterIP: None
  selector:
    app: kafka
    instance: {{ .Name }}
```

## Testing your operator

You should aim for your operators being tested for day 1. To help you with testing your operator, we have developed a tool called test harness (it's also what we use to test KUDO itself). For more information please go to [test harness documentation](docs/testing.md).
