---
kep-number: 24
title: Enhanced Operator Parameters
short-desc: Parameter types other than `string`
authors:
  - "@nfnt"
owners:
  - "@nfnt"
creation-date: 2020-02-21
last-updated: 2020-03-06
status: implemented
---

# Enhanced Operator Parameters

## Table of Contents

- [Enhanced Operator Parameters](#enhanced-operator-parameters)
  - [Table of Contents](#table-of-contents)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [Support Optional Parameter Types](#support-optional-parameter-types)
    - [Provide Parameter Validation](#provide-parameter-validation)
    - [Notes](#notes)
  - [Alternatives](#alternatives)
    - [Convert YAML in string to dictionaries](#convert-yaml-in-string-to-dictionaries)
    - [Allow Arbitrary Parameter Values](#allow-arbitrary-parameter-values)
  - [Implementation History](#implementation-history)

## Summary

This KEP aims to improve the use-cases that can be covered by parameters. By supporting additional types, templates can access lists and dictionaries. Furthermore, validation can be implemented for known types.

## Motivation

Currently only string parameters are supported by KUDO. This limits the possible use-cases of parameters. Various use cases benefit from having parameters that are lists or dictionaries. While some of this functionality can be implemented in Go templates using [Sprig functions][1], this is unintuitive.
Having parameter types creates the possibility for parameter validation.

### Goals

- Provide operator developers with more versatile parameters
- Allow the definition of validation functions for parameters

### Non-Goals

- Change existing parameter definitions in `params.yaml`

## Proposal

### Support Optional Parameter Types

By adding an optional `type` field to the list of parameters, parameter types can be specified. If this field isn't present, it defaults to the `string` type. Other types like `list`, `dict` or `integer` are possible. KUDO's renderer uses the type information to convert the string value to the respective type. For example, for the `dict` type, the input value would be YAML that is unmarshalled to a `map[string]interface{}`. Go's template engine allows to access and iterate list and dictionary items.

Consider a cluster that has nodes in multiple regions. We want to ensure that a nginx deployment has 2 replicas in one region and 1 replica in the other one. Furthermore they should support both HTTP as well as HTTPS, hence need a list of port definitions. These parameters are described by `list` and `dict` types:

```yaml
parameters:
  - name: ports
    description: Ports
    type: list
    default: "[ 80, 443 ]"
  - name: topology
    description: Node topology
    type: dict
    default: |
      - region: us-east-1
        replicas: 2
      - region: us-west-1
        replicas: 1
```

The deployment template then iterates over the items of the respective parameters:

```yaml
{{ range .Params.topology }}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app.kubernetes.io/name: nginx
spec:
  replicas: {{ .replicas }}
  selector:
    matchLabels:
      app.kubernetes.io/name: nginx
  template:
    metadata:
      name: nginx
      labels:
        app.kubernetes.io/name: nginx
    spec:
      nodeSelector:
        topology.kubernetes.io/region: {{ .region }}
      containers:
      - name: nginx
        image: nginx:stable
        ports:
        {{ range $.Params.ports }}
        - containerPort: {{ . }}
        {{ end }}
{{ end }}
```

### Provide Parameter Validation

Adding types for lists and dictionaries provides access to the items of these structures in templates. Adding types like integers won't provide benefits for templating, but would allow for input value validation. KUDO could, e.g., validate that an integer isn't too large, that a list doesn't have too many entries, or that a string isn't too long. Adding parameter validation is independent from adding parameter types.

Adding validation rules to parameters could look like this:

```yaml
parameters:
  - name: replicas
    description: Number of replicas
    type: integer
    minimum: 1
    maximum: 3
  - name: instance-name
    description: Name of the instance
    regex: ^\w-_*$
    max-length: 40
```

### Notes

Setting parameters from the KUDO CLI needs to add support for YAML value input. In addition to multi-line strings, the CLI should also allow to use files and heredocs as parameter value input. See [draft KEP-26](https://github.com/kudobuilder/kudo/pull/1364).

## Alternatives

### Convert YAML in string to dictionaries

A template function `fromYAML` could be used to convert a parameter value to a dictionary. The example above would then change to

```yaml
{{ $topology := (fromYAML .Params.topology) }}
{{ range $topology }}
...
{{ end }}
```

In KUDO, this function would work the same way as the `dict` type conversion mentioned above. The difference is that the conversion has to be explicitly done by the operator developer in the template. Also, this approach wouldn't allow for parameter validation in KUDO.

### Allow Arbitrary Parameter Values

Support for lists or dictionaries could be added by treating every parameter value as YAML. Because a single string is valid YAML, this would be compatible with string parameters. Though, this approach creates problems with Go template pipelines. Some template functions like `eq` no longer work with parameters that have been converted from YAML, because their converted type is no longer `string` but could be `int`, `float`, or `boolean`.

## Implementation History

- 2020/02/21 - Initial draft. (@nfnt)
- 2020/03/06 - [Implemented](https://github.com/kudobuilder/kudo/pull/1376). (@nfnt)

[1]: http://masterminds.github.io/sprig/
