# Update and Upgrade Plans


This demo uses a different toy framework described in the `../config/samples/upgrade.yaml` file.  There are two `Framework` versions defined for this `Framework`, and each `FrameworkVersion` has three plans: `deploy`, `update` and `upgrade`.


# Deploy Plan

The deploy plan is automatically run for new instances:

```bash
$ kubectl apply -f config/samples/upgrade.yaml
framework.kudo.k8s.io/upgrade created
frameworkversion.kudo.k8s.io/upgrade-v1 created
frameworkversion.kudo.k8s.io/upgrade-v2 created
instance.kudo.k8s.io/up created
```

The PlanExecution object that gets created gets suffixed with a timestamp for uniqueness.

```bash
$ kubectl get planexecutions -l instance=up
NAME                  AGE
up-deploy-773729000   33s
```

And the logs:

```bash
$ kubectl logs jobs/up-deploy-job
deploy for v1
Going to sleep for 15 seconds
```

# Update the Instance

An upgrade of the instance is run when the Spec of the Instance is changed, but the FrameworkVersion remains the same:

```
kubectl patch instance up -p '{"spec":{"parameters":{"SLEEP":"60"}}}' --type=merge
instance.kudo.k8s.io/up patched
```

```bash
kubectl get planexecutions -l instance=up
NAME                  AGE
up-deploy-773729000   16m
up-update-129951000   40s
```

```bash
$ kubectl logs jobs/up-update-job
update for v1
Going to sleep for 60 seconds
```

# Upgrade

Upgrades occur when the `FrameworkVersion` is changed.  The Upgrade from the NEW `FrameworkVersion` is executed:


```bash
$  kubectl patch instance up -p '{"spec":{"frameworkVersion":{"name":"upgrade-v2"}}}' --type=merge
instance.kudo.k8s.io/up patched
$ kubectl get planexecutions -l instance=up
NAME                   AGE
up-deploy-539794000    3m
up-update-24526000     2m
up-upgrade-142970000   5s
$ kubectl get jobs
NAME                      COMPLETIONS   DURATION   AGE
up-deploy-job             1/1           17s        3m58s
up-update-job             1/1           62s        3m9s
up-upgrade-job            0/1           58s        58s
$ kubectl logs job/up-upgrade-job
upgrade for v2
Going to sleep for 60 seconds
```