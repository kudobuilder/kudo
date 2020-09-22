---
kep-number: 25
title: Template toYaml function
short-desc: New toYaml function for use in templates
authors:
  - "@porridge"
owners:
  - "@porridge"
editor: @porridge
creation-date: 2020-02-25
last-updated: 2020-03-02
status: implemented
see-also:
  - KEP-24
---

# Template toYaml function

## Table of Contents

* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Proposal](#proposal)
    * [User Stories](#user-stories)
    * [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    * [Risks and Mitigations](#risks-and-mitigations)
* [Implementation History](#implementation-history)
* [Alternatives](#alternatives)
    * [Deep transcription field-by-field](#deep-transcription-field-by-field)

## Summary

This KEP describes addition of a function called `toYaml` for use in the operator templates.

## Motivation

This enhancement will make it possible for operator developers to allow users to
pass arbitrary structured parameters and directly embed them into resources.

### Goals

- Make it possible for an operator developer to use a `toYaml` function in their templates
on a `.Params` value.

### Non-Goals

- Make it possible for an operator user to provide a parameter with value type other
than `string`. We rely on [KEP-24](https://github.com/kudobuilder/kudo/pull/1356) to provide
this functionality.
- Make it possible for a user to provide a complex structured parameter value in
typical multi-line YAML syntax in a convenient way.
This will be addressed by a separate KEP. For now, a user will need to rely on
passing the value via a `--parameter` command line flag using JSON syntax
(which is a subset of YAML).

## Proposal

The idea is to provide an analog of the `toYaml`
[function provided by Helm](https://github.com/helm/helm/blob/be1e974cccec4f5583ef6e67b229f35f9e6edd2e/pkg/chartutil/files.go#L168-L179).
Sadly, I could not find any documentation for it, but it is [used frequently in Helm charts](https://github.com/helm/charts/search?q=toYaml&unscoped_q=toYaml).

### User Stories

There were at least a couple of feature requests that would be solved by this
enhancement. They were mostly concerned with configuring some aspects of k8s pods:
- [#1221](https://github.com/kudobuilder/kudo/issues/1221): `nodeSelector`, `podAnnotations`, `resources` and `podAffinity`
- [#951](https://github.com/kudobuilder/kudo/issues/951) `nodeAffinity` and `tolerations`

The common theme is that these k8s features requires deeply nested YAML syntax
([some examples](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#an-example-of-a-pod-that-uses-pod-affinity) are 5 levels deep!)

Here is how using this might look like:

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    metadata:
    {{- if or .Params.podAnnotations .Params.enableMetrics }}
      annotations:
      {{- if .Params.enableMetrics }}
        prometheus.io/scrape: "true"
      {{- end }}
      {{- if .Params.podAnnotations }}
{{ toYaml .Params.podAnnotations | trim | indent 8 }}
      {{- end }}
    {{- end }}
```

This matches the idiomatic way to do this in Helm.
Here is some [context](https://github.com/kudobuilder/kudo/pull/1363#discussion_r386247611)
on the need for `trim` and `indent`.

As mentioned in the [Non-Goals](#non-goals) section, a user would initially
pass the value for above example `podAnnotations` parameter like this:

```bash
kubectl kudo install theoperator --parameter='{"anntation.io": "somevalue"}'
```

### Implementation Details/Notes/Constraints

This proposal relies on [KEP-24](https://github.com/kudobuilder/kudo/pull/1356) to provide
support for parameter values other than `string`, as this function would be nearly
useless without it. However the author of KEP-24 [decided to keep it separate](https://github.com/kudobuilder/kudo/pull/1356/files#r382593155).


### Risks and Mitigations

Being able to just dump a deeply nested data structure in the middle of a template
adds a lot of flexibility, but could also open up opportunities for abuse. In the extreme
case, one can imagine a template consisting only of a single line:

```yaml
{{ .Params.resource }}
```

This would allow the user to pass via the `--parameter` flag a complete resource
definition of arbitrary name and type. Since changes of this resource
between `kubectl kudo` invocations would be unbounded, it would most likely have unintended
consequences (including KUDO controller losing track of created resources).

Ultimately, operator developers need to apply this feature with some care.

## Implementation History

- 2020-02-25 - Initial draft. (@porridge)
- 2020-02-28 - Changed to implementable. (@porridge)
- 2020-03-02 - Implemented in [#1375](https://github.com/kudobuilder/kudo/pull/1375). (@porridge)

## Alternatives

### Deep transcription field-by-field

In the absence of a `toYaml` function, in order to provide a way to e.g. customize
a `podAffinity` setting, the operator developer would need to hand-craft a complex
template that would walk the deep data structure provided by a user and
re-create it again in YAML.

In case of such deeply-nested structures, this template would be very hard to read and
error-prone to edit.
