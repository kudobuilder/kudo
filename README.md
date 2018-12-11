# Maestro

Maestro provides a declarative approach to building production-grade Kubernetes Operators covering the entire application lifecycle. 

## Installation Instructions

- [Configure kubectl ](https://kubernetes.io/docs/tasks/tools/install-kubectl/) 
- Get maestro repo: go get github.com/maestrosdk/maestro/
- `cd $GOPATH/src/github.com/maestrosdk/maestro`
- `make install` to deploy universal CRDs
- `make run` to run the Operator with local go environment


## Deploy your first Application

Create a `Framework` object for Zookeeper
```bash
$ kubectl apply -f config/samples/zookeeper-framework.yaml
framework.maestro.k8s.io/zookeeper created
```

Create a `FrameworkVersion` for the Zookeeper  `Framework`

```bash
$ kubectl apply -f config/samples/zookeeper-frameworkversion.yaml
frameworkversion.maestro.k8s.io/zookeeper-1.0 created
```
 

Create an Instance of the Zookeeper
```
$ kubectl apply -f config/samples/zookeeper-instance.yaml
instance.maestro.k8s.io/zk created
```

When an instance is created, the default `deploy` plan is executed

```
$ kubectl get planexecutions
NAME                  AGE
zk-deploy-317743000   53s
```

The statefulset defined in the `FrameworkVersion` comes up with 3 pods:

```bash
kubectl get statefulset zk-zk
NAME    DESIRED   CURRENT   AGE
zk-zk   3         3         1m20s
```

```bash
 kubectl get pods
NAME                    READY   STATUS             RESTARTS   AGE
zk-zk-0                 1/1     Running            0          23s
zk-zk-1                 1/1     Running            0          23s
zk-zk-2                 1/1     Running            0          23s
```


## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](http://slack.k8s.io/)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-dev)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

[owners]: https://git.k8s.io/community/contributors/guide/owners.md
[Creative Commons 4.0]: https://git.k8s.io/website/LICENSE
