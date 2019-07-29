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

You can see that the step `nginx` references `deployment.yaml`. KUDO automatically expects this file to exist inside `templates` folder. So let’s create `templates/deployment.yaml`.

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

