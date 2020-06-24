---
kep-number: 22
title: Diagnostics Bundle
short-desc: Automatic collection of diagnostics data for KUDO operators
authors:
  - "@mpereira"
  - "@vemelin-epm"
owners:
  - "@mpereira"
  - "@gerred"
  - "@zen-dog"
  - "@vemelin-epm"
creation-date: 2020-01-24
last-updated: 2020-03-26
status: provisional
---

# [KEP-22: Diagnostics Bundle](https://github.com/kudobuilder/kudo/issues/1152)

## Table of contents

- [Summary](#summary)
- [Prior art, inspiration, resources](#prior-art-inspiration-resources)
  - [Diagnostics](#diagnostics)
- [Concepts](#concepts)
  - [Fault](#fault)
  - [Failure](#failure)
  - [Operator](#operator)
  - [Operator instance](#operator-instance)
  - [Application](#application)
  - [Operator developer](#operator-developer)
  - [Operator user](#operator-user)
  - [Diagnostic artifact](#diagnostic-artifact)
- [Goals](#goals)
  - [Functional](#functional)
  - [Non-functional](#non-functional)
- [Non-goals](#non-goals)
- [Requirements](#requirements)
- [Proposal](#proposal)
  - [Operator user experience](#operator-user-experience)
  - [Operator developer experience](#operator-developer-experience)
- [Resources](#resources)
- [Implementation history](#implementation-history)

## Summary

Software will malfunction. When it does, data is needed so that the problem can be
diagnosed, mitigated in the short term, and fixed for the long term. This KEP
is about creating programs that will automatically collect data and store them
in an easily distributable format.

These programs must be easy to use, given that they will potentially be used in
times of stress where faults or failures have already occurred. Secondarily, but
still importantly, these programs should be easily extensible so that the
collection of data related to new fault types can be quickly implemented and
released.

Applications managed by KUDO operators are very high in the stack (simplified
below):

| Layer               | Concepts                                                                                   |
| ------------------- | ------------------------------------------------------------------------------------------ |
| Application         | (Cassandra's `nodetool status`, Kafka's consumer lag, Elasticsearch's cluster state, etc.) |
| Operator instance   | (KUDO plans, KUDO tasks, etc.)                                                             |
| KUDO                | (controller-manager, k8s events, logs, objects in kudo-system, etc.)                       |
| Kubernetes workload | (Pods, controllers, services, secrets, etc.)                                               |
| Kubernetes          | (Docker, kubelet, scheduler, etcd, cloud networking/storage, Prometheus metrics, etc.)     |
| Operating system    | (Linux, networking, file system, etc.)                                                     |
| Hardware            |                                                                                            |

These layers aren't completely disjoint. This KEP will mostly focus on:

- Application
- Operator instance
- KUDO
- Kubernetes workload

## Prior art, inspiration, resources

### Diagnostics

1.  [replicatedhq/troubleshoot](https://github.com/replicatedhq/troubleshoot)

    Does preflight checks, diagnostics collection, and diagnostics analysis for
    Kubernetes applications.

2.  [mesosphere/dcos-sdk-service-diagnostics](https://github.com/mesosphere/dcos-sdk-service-diagnostics/tree/master/python)

    Does diagnostics collection for
    [DC/OS SDK services](https://github.com/mesosphere/dcos-commons).

    Diagnostics artifacts collected:

    - Mesos-related (Mesos state)
    - SDK-related (Pod status, plans statuses, offers matching, service
      configurations)
    - Application-related (e.g., Apache Cassandra's
      [nodetool](http://cassandra.apache.org/doc/latest/tools/nodetool/nodetool.html)
      commands, Elasticsearch's
      [HTTP API](https://www.elastic.co/guide/en/elasticsearch/reference/current/rest-apis.html)
      responses, etc.)

3.  [dcos/dcos-diagnostics](https://github.com/dcos/dcos-diagnostics)

    Does diagnostics collection for [DC/OS](https://dcos.io/) clusters.

4.  [mesosphere/bun](https://github.com/mesosphere/bun)

    Does diagnostics analysis for archives created with `dcos/dcos-diagnostics`.
    
5.  [vmware-tanzu/sonobuoy](https://github.com/vmware-tanzu/sonobuoy)

    A tool to understand the cluster state by collecting data from:
    - Kubernetes conformance tests
    - resources info
    - custom plugins

It is also important to notice that some applications have existing tooling
for application-level diagnostics collection, either built by the supporting
organizations behind the applications or the community. A few examples:

- [Elasticsearch's support-diagnostics](https://github.com/elastic/support-diagnostics)
- [Apache Kafka's System Tools](https://cwiki.apache.org/confluence/display/KAFKA/System+Tools)

## Concepts

### Fault

One component of the system deviating from its specification.

### Failure

The system as a whole stops providing the required service to the user.

### Operator

A KUDO-based
[Kubernetes Operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/),
e.g. [kudo-cassandra](https://github.com/mesosphere/kudo-cassandra-operator),
[kudo-kafka](https://github.com/mesosphere/kudo-kafka-operator).

### Operator instance

A KUDO `Instance` resource which gets created as a result of `kubectl kudo install`.

### Application

Underlying software that is managed by an operator instance, e.g., Apache
Cassandra, Apache Kafka, Elasticsearch, etc.

### Operator developer

Someone who builds and maintains operators.

### Operator user

Someone who installs and maintains operator instances.

### Diagnostic artifact

A file, network response, or command output that contains information that is
potentially helpful for operator users to diagnose faults with their operator
instances, and for operator users to provide to operator developers and/or
people who support operators.

## Goals

### Functional

- Collect "Kubernetes workload"-specific diagnostics artifacts related to an
  operator instance
- Collect KUDO-specific diagnostics artifacts related to an operator instance
- Collect application-specific diagnostics artifacts related to an operator
  instance
- Bundle all diagnostics artifacts into an archive

### Non-functional

1.  Provide an **easy** experience for operator users to collect diagnostic
    artifact archives
2.  Be resilient to faults and failures. Collect as much diagnostics artifacts
    as possible and allow failed collections to be retried (idempotency) and
    incremented to (like `wget`'s `--continue` flag), in a way that collection
    is _resumable_
3.  Incorporate standard tools that are already provided by either organizations
    or the community behind applications as much as possible

## Non-goals

- Collection of Kubernetes-related diagnostics artifacts that are not related to diagnosing issues with KUDO operators/instances.
- At least not initially: collection of metrics from monitoring services (e.g.,
  Prometheus, Statsd, etc.).
- Automatic fixing of faults
- Preflight checks e.g. checking if the cluster is in a state that allows an operator installation
- Analysis of collected artifacts
- Extending diagnostics bundle with custom artifact collectors by the *operator user*

## Requirements

- MUST create an archive with diagnostics artifacts related specifically to an
  operator instance
- MUST include application-related diagnostics artifacts in the archive
- MUST include instance-related diagnostics artifacts in the archive
- MUST include KUDO-related diagnostics artifacts in the archive
- MUST include "Kubernetes workload"-related diagnostics artifacts in the
  archive
- MUST accept parameters and work without interactive prompts
- SHOULD work in airgapped environments
- SHOULD report the versions of every component and tool, in the archive (e.g.,
  the version of the collector, the application version, the operator version,
  the KUDO version, the Kubernetes version, etc.)
- SHOULD follow Kubernetes' ecosystem conventions and best practices
- MUST be published as a static binary
- SHOULD make it possible to publish archive to cloud object storage (AWS S3,
  etc.)
- MUST follow SemVer

## Proposal

### Operator user experience

The output from diagnostics collection is an archive containing all
diagnostics artifacts for the provided operator instance.

```bash
kubectl kudo diagnostics collect --instance=%instance% --namespace=%namespace%
```

### Operator developer experience

To configure diagnostics for a given operator version, this KEP introduces an optional top-level
`diagnostics` key in operator.yaml.

#### Diagnostics collection

The following diagnostics will be implicitly collected without any configuration
from the operator developer:

- Logs for deployed pods related to the KUDO Instance
- YAML for created resources, including both spec and status, related
  to the KUDO instance
- Output of `kubectl describe` for all deployed resources related to the KUDO
  Instance
- Current plan status, if one exists, or the KUDO Instance
- Information about the KUDO instance's Operator and OperatorVersion
- Logs for the KUDO controller manager
- Describe for the KUDO controller manager resources
- RBAC resources that are applicable to the KUDO controller manager
- Current settings and version information for KUDO
- k8s events (can we filter them for resources that the instance owns?)

Operator developer experience, then, focuses on customizing diagnostics
information to gather information about the running application. The following
forms are available, subject to change over time:

- **Copy**: Copy a file out of a running pod. This is useful for non-stdout/err
  logs, configuration files, and other artifacts generated by an application.
  Higher level resources can also be used, which will copy the file on all pods
  selected by that resource.
- **Command**: Run a command on a running pod and copy the stdout/err. Higher level
  resources can also be used, which will run the command on all pods selected by
  that resource.
- **Request**: Make a request to a named service and port and copy the result of the request.

While some of these are redundant (_Request_ can be a command or job), the intent
is to provide a high level experience where possible so that operator developers
don't necessarily need to maintain a `curl` container as part of their
application stack.

Operator-defined diagnostics collection is defined in a new `diagnostics.bundle.resources`
key in `operator.yaml`:

```yaml
diagnostics:
  bundle:
    resources:
      - description: Zookeeper Configuration File
        name: zookeeper-configuration
        kind: Copy
        spec:
          path: 
            - /opt/zookeeper/server.properties
            - /opt/zookeeper/zoo.cfg
          selector:
            matchLabels:
              app: zookeeper
              heritage: kudo
      - description: DNS information for running pod
        name: dns-information
        kind: Command
        spec:
          command: # Can be string or array
            - nslookup
            - google.com
          selector:
            matchLabels:
              app: zookeeper
              heritage: kudo
      - description: Zookeeper Client Service Status
        name: cs-stat
        kind: Request
        spec:
          serviceRef:
            name: zookeeper-instance-cs
            port: client
          query: stat
    redact:
      - description: Authentication information
        spec:
          regex: "^host: %w+$"
```

This key is **OPTIONAL**. Default diagnostics collection will happen regardless
of the `diagnostics.bundle` key's presence. Future iterations of this might reduce the
complexity of selecting resources to run commands and files on.

_Redaction_ is an important part of diagnostics collection. It enables diagnostics
to be portably sent to third parties that should not have sensitive information
that logs and files can contain.

By default, KUDO redacts all resources (and custom resources) to obscure values
contained within the KUDO Instance's secrets. This is configurable with the
`diagnostics.redactSecrets` key.

There may be other fields that need to be redacted. To solve for this, KUDO
introduces the `diagnostics.bundle.redact` key in `operator.yaml`, which
contains a list of filters that files pass through before writing to disk.
Custom filters use either a regular expression or an object reference and
JSONPath to derive the values to be redacted.

All redacted values appear as `**REDACTED**` in relevant logs and files.

### More Notes

- Do we need to introduce a notion of the collector or controller manager
  signing and/or encrypting bundles? TBD.

## Resources

### bundle.resources

An individual bundle resource is represented as an element in the list inside of the
`diagnostics.bundle.resources` key. Resources have the following REQUIRED keys:

- **name**: The machine-readable name of the file. This is used for both
  references (if needed in the future) and filenames. Extension is OPTIONAL,
  but may be useful for inferring MIME types.
- **kind**: The kind of bundle item.
- **spec**: The attributes of a particular kind. This is different for every
  kind.  
  
and an OPTIONAL key
- **description**: The human-readable name of the file.

Also, specs may include a `selector` that conforms to Kubernetes [selector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)
and allows to select a set of resources. The individual filters defined in this key are `AND`-ed to form the resulting filter.
The key has AT LEAST ONE of the following sub-keys:

- **matchLabels**: key-value pairs of labels that should be defined on a resource to be processed.
- **matchExpressions**: set-based requirements (if supported by the resource)

### bundle.resources.Copy

- **path**: Absolute path inside of the referenced pods.
- **selector**

### bundle.resources.Command

- **command**: Command to run. May be a string or an array.
- **selector**

### bundle.resources.Request

- **serviceRef**: Object containing references to a Kubernetes service. This is
  scoped to services made by KUDO.
- **serviceRef.name**: Name of the service.
- **serviceRef.port**: Name of the service port. MUST be a named port, not an
  integer value.
- **urlPath**: URL path
- **method** a _safe_ method for HTTP request, absent for plain TCP.
- **query** query part of the URL or a TCP payload (e.g. Zookeeper "Four Letter Word")

### bundle.redact

`bundle.redact` key comprises a list of filters to find and redact sensitive data. Each redaction filter contains the following keys:

- **name**: The human readable name of the filter.
- **regex** (optional): Regular expression, not encased in slashes, to use.
  Regex flags are not supported, and are global and case sensitive.
- **objectRef** (optional): Required if `jsonPath` is present.
- **jsonPath** (optional): JSONPath referencing a non-object key in the
  referenced object. All instances of the value of this key will be removed and
  replaced with `**REDACTED**`. Required if `objectRef` is present.

## Implementation history

- 2020/01/24 - Initial draft (@mpereira)
- 2020/02/11 - Support bundles info (@gerred)

