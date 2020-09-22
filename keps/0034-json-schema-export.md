---
kep-number: 34
short-desc: JSON-schema export
title: JSON-schema export
authors:
  - "@aneumann82"
owners:
  - "@aneumann82"
creation-date: 2020-09-22
last-updated: 2020-09-22
status: provisional
see-also:
  - KEP-33
---

# JSON-schema export

## Table of Contents

<!--ts-->
<!--te-->

## Summary

Extend the existing `params.yaml` with metadata and support for an export of a simple JSON-schema.

## Motivation

JSON-schema is the preferred way to generate a user-friendly UI to configure a new instance of a KUDO operator or to trigger plans with parameters.  KUDO parameters should be described by a JSON-schema.

KEP-33 proposes a full replacement of the existing flat list `params.yaml` with a nested and typed structure described by JSON-schema. This is a big breaking change and requires a lot of work to get right in a single release.

This KEP proposes to enhance the flat list parameters with additional attributes and add support to export a JSON-schema. This is a first step towards KEP-33

TODO: How 

### Goals

- Add additional attributes required for a useful JSON-schema
- Add support to export a JSON-schema that describes the parameters
- Maybe: Allow installation/update of an operator instance with a structured object described by the exported JSON-schema? 
- Prepare Implementation of KEP-33 in the future 

### Non-Goals

- Handle multiple CRD versions
- Replace the `params.yaml` with a json-schema
- Add validation for parametersKEP-34

## Proposal

This is where we get down to the nitty gritty of what the proposal actually is.

### User Stories [optional]

Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of the system.
The goal here is to make this feel real for users without getting bogged down.

#### Story 1

#### Story 2

### Implementation Details

Additional Attributes 
- `group` string - A top-level group to organize parameters. If a parameter does not specify a group, it gets assigned to a "default" group.
- `advanced` boolean - A flag that indicates that this parameter is for advanced configuration and can be hidden by default. Defaults to "false"
- `hint` string - A short description about the syntax of this parameter

Additional Types:
- `boolean` A "true" or "false" value, from the JSON "true" or "false" productions
- `number` An arbitrary-precision, base-10 decimal number value, from the JSON "number" production

```yaml
apiVersion: kudo.dev/v1beta1
parameters:
  - name: ZOOKEEPER_URI
    displayName: "zookeeper cluster URI"
    description: "host and port information for Zookeeper connection. e.g. zk:2181,zk2:2181,zk3:2181"
    default: "zookeeper-instance-zookeeper-0.zookeeper-instance-hs:2181"
    group: "Zookeeper"
    required: true
```

### Risks and Mitigations

What are the risks of this proposal and how do we mitigate.
Think broadly.
For example, consider both security and how this will impact the larger kubernetes ecosystem.

## Graduation Criteria

How will we know that this has succeeded?
Gathering user feedback is crucial for building high quality experiences and owners have the important responsibility of setting milestones for stability and completeness.
Hopefully the content previously contained in [umbrella issues][] will be tracked in the `Graduation Criteria` section.

[umbrella issues]: https://github.com/kubernetes/kubernetes/issues/42752

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling owner acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first KUDO release where an initial version of the KEP was available
- the version of KUDO where the KEP graduated to general availability
- when the KEP was retired or superseded

## Drawbacks [optional]

Why should this KEP _not_ be implemented.

## Alternatives [optional]

Similar to the `Drawbacks` section the `Alternatives` section is used to highlight and record other possible approaches to delivering the value proposed by a KEP.

## Infrastructure Needed [optional]

Use this section if you need things from the project/owner.
Examples include a new subproject, repos requested, github details.
Listing these here allows an owner to get the process for these resources started right away.
