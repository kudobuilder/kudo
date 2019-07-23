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
