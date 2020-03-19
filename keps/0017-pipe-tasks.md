---
kep-number: 17
title: Pipe Tasks
short-desc: Feature for a mechanism that allows piping of resources (files) from one task to another
authors:
  - "@zen-dog"
owners:
  - "@zen-dog"
editor: TBD
creation-date: 2019-09-12
last-updated: 2019-09-12
status: implemented
---

# Pipe Tasks

## Table of Contents

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Proposal](#proposal)
    * [Limitations](#limitations)
    * [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints-optional)
* [Alternatives](#alternatives)

## Summary

Developing complicated operators often require generating files in one task and reusing them in a later one. A common example is generating custom certificates/dynamic configuration files in the bootstrap task and using them in the later deployment step of the service. This KEP describes how KUDO could help operator developers by making this task seamless and automated.

## Motivation

We want to ease the development of operators with complicated life-cycles by enabling to pipe files from one task to another. While this can be achieved in a manual way, involving custom scripts, this is exactly the kind of "magic glue" that KUDO could provide to the operator developers to ease the development process.

### Goals

The goal of this KEP is to describe application examples, limitations and ways to implement passing files between tasks. We aim to help operator developers pipe small files (<1Mb) like certificates and dynamically generated configuration which can be referenced later as ConfigMaps and Secrets.

### Non-Goals

Allowing to pipe all kind of files (>1Mb) between tasks requires a general-purpose storage solution (e.g. S3) available in the cluster. While certainly possible, this is not the goal of this KEP.

## Proposal

This section describes how pipe tasks and files they produce are configured in the operator.This proposal is currently limited to pipe tasks which create files which are assigned to a key in a ConfigMap or a Secret. Here is a pipe task definition that produces a file that will be stored as a Secret:
```yaml
tasks:
  - name: gencert
    kind: Pipe
    spec:
      pod: cert-pod.yaml
      pipe:
        - file: /usr/share/MyKey.key
          kind: Secret # or a ConfigMap
          key: Mycertificate
``` 

`pod` field is described in detail below. `key` will be used by in the template file to reference generated artifact e.g:
 ```yaml
volumes:
- name: cert
    secret:
      secretName: {{ .Pipes.Mycertificate }}
```
will create as a volume from the generated secret.

In the above example we would create a secret named `instancemyapp.deploy.bootstrap.gencert.mycertificate` which captures instance name along with plan/phase/step/task of the secret origin. The secret name would be stable so that a user can rerun the certificate generating task and reload all pods using it. Pipe file name is used as Secret/ConfigMap data key. Secret/ConfigMap will be owned by the corresponding Instance so that artifacts are cleaned up when the Instance is deleted. `pipe` field is an array and can define how multiple generated files are stored and referenced.

The corresponding `gencert` task can be used as usual in e.g.:
```yaml
plans:
  deploy:
    strategy: serial
    phases:
     - name: bootstrap
       strategy: serial
       steps:
        - name: gencert
          tasks:
           - gencert
```

Note that piped files has to be generated before they can be used. In the example above, `bootstrap` phase has a strategy `serial` so that certificate generated in the `gencert` step can be used in subsequent steps. Or stated differently resources can not reference piped secrets/configmaps generated within the same step or within a parallel series of steps (it has to be a different step in the phase with serial strategy or a different phase). 

Pipe task's `spec.pod` field must reference a [core/v1 Pod](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#pod-v1-core) template. However, there are limitations. Reasons for that are explained below in the implementation details. In a nutshell:
- a pipe pod should generate artifacts in its init container
- it has to define and mount an emptyDir volume (where its generated files are stored) 

```yaml
apiVersion: v1
kind: Pod
spec:
  volumes:
  - name: shared-data
    emptyDir: {}

  initContainers:
    - name: init
      image: busybox
      command: [ "/bin/sh", "-c" ]
      args:
        - openssl req -new -newkey rsa:4096 -x509 -sha256 -days 365 -nodes -out MyCertificate.crt -keyout /usr/share/MyKey.key
      volumeMounts:
        - name: shared-data
          mountPath: /tmp 
```

Any subsequent step resource (if the phase strategy is `serial`) might reference previously generated file by its key e.g.:
```yaml
# some Pod spec referenced in a subsequent step
spec:
  containers:
  - name: myapp
    image: myapp:latest
    volumeMounts:
    - name: cert
      mountPath: /еtc/certs
  volumes:
  - name: cert
    secret:
      secretName: {{.Pipes.Mycertificate}}
```

### Limitations
- File generating Pod has to be side-effect free (meaning side-effects that are observable outside of the container like a 3rd party API call) as the container might be executed multiple times on failure. A `restartPolicy: OnFailure` is used for the pipe pod.
- Only files <1Mb are applicable to be stored as ConfigMap or Secret. A pipe task will fail should it try to copy files >1Mb

### Implementation Details/Notes/Constraints
There are several ways to implement pipe tasks, each one having its challenges and complexities. The approach below allows us not to worry about Pod container life-cycle as well as keep the storing logic in the KUDO controller:
- Provided Pod is enriched with a main container, which uses a `busybox` image, running the `sleep infinity` command, which purpose is to wait for KUDO to extract and store the files.
- Generating files in the initContainer is the simplest way to wait for container completion. A pipe pod status: `READY: 1/1, STATUS: Running` means that the init container has run successfully. As this point KUDO can copy out referenced files using `kubectl cp` and store them as specified. 
- Pod init container can not have Lifecycle actions, Readiness probes, or Liveness probes fields defined which simplifies implementation significantly
- Once files are stored, KUDO can delete the pipe pod and proceed to the next task.

Here is a minimal example demonstrating the proposed implementation:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: pipe-task
spec:
  volumes:
  - name: shared-data
    emptyDir: {}

  # Inject provided container generating /tmp/foo.txt
  initContainers:
    - name: init
      image: busybox
      command: [ "/bin/sh", "-c" ]
      args:
        - echo "foo-bar-bazz" > /tmp/foo.txt
      volumeMounts:
        - name: shared-data
          mountPath: /tmp

  # Wait for KUDO controller to copy out the file
  containers:
    - name: sleep
      image: busybox
      command: [ "/bin/sh", "-c" ]
      args: [ "sleep infinity" ]
      volumeMounts:
        - name: shared-data
          mountPath: /tmp
  restartPolicy: OnFailure
```

The generated file can be copied out of the Pod with:
```bash
$ kubectl cp default/pipe-task:/tmp/foo.txt /dev/stdout
foo-bar-bazz
```

## Alternatives

An alternative approach would allow user to specify the main container (`containers` field) or let the user provide a complete Pod spec. We would inject a sidecar with a go executable which have to:
- Use controller runtime, watch its own Pod status and wait for termination of the main container
- Once the main container exits, it would copy the referenced files and store them as specified
- We would use `restartPolicy: Never/OnFailure` to prevent the main container from restarting

While this approach would allow users to specify complete Pods as pipe task resources, a sidecar implementation would be an additional source of complexity and failure potential. The best code is no code at all.
