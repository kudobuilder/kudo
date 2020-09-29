---
kep-number: 35
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

JSON-schema is the preferred way to generate a user-friendly UI to configure a new instance of a KUDO operator or to trigger plans with parameters. KUDO should be able to export a JSON-schema that describes the required parameters.

KEP-33 proposes a full replacement of the existing flat list `params.yaml` with a nested and typed structure described by JSON-schema. This is a big breaking change and requires a lot of work to get right in a single release.

This KEP proposes to enhance the flat list parameters with additional attributes and add support to export a JSON-schema. This is a first step towards KEP-33

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

### User Stories

#### Story 1

A user wants to generate a UI to install a KUDO operator. He wants to use an automatic generator, for example https://rjsf-team.github.io/react-jsonschema-form/

### Implementation Details

#### Additional Types: 

To bring the parameters up to par with JSON-schema, we add these additional types:
- `boolean` A "true" or "false" value, from the JSON "true" or "false" productions
- `number` An arbitrary-precision, base-10 decimal number value, from the JSON "number" production

#### Additional Attributes

JSON-Schema already provides good support for generated UIs, but it can be enhanced. Additionally, we do not want to adopt the full schema yet, there is a different KEP for that.

All of the additional attributes are fully optional.

`group` 
An attribute of type "string". This attribute describes a top-level group used for this parameter.

`advanced` 
An attribute of type "boolean" that indicates that this parameter is for advanced configuration and can be hidden by default. Defaults to "false". Parameters that have "advanced=true" *must* have a default value, as they might not be visible at first

`hint`
An attribute of type "string" that should provide a short description about the syntax of this parameter. Usually shown below the actual input element.

`enum` 
An array of values that are allowed to be used. Each entry must be of the type defined by the `type` attribute. If a default is specified, it must be listed as well.

```yaml
  - name: NODE_DOCKER_IMAGE_PULL_POLICY
    title: "Docker Image Pull Policy"
    hint: "Cassandra node Docker image pull policy"
    description: "The Docker image pull policy. Can be changed, for example if the image is pre-loaded on the Kubernetes nodes."
    default: "Always"
    type: "string"
    enum: 
      - "Always"
      - "Never"
      - "IfNotPresent"
    group: "Node"
    advanced: true
    required: true
```

### JSON-Schema export

Extend the `kudo package list parameters` with an option `--output json-schema` to describe the parameters as JSON-Schema.

Alternatively, add a new command `kudo package list json-schema`, which then could be used with `--output json` or `--output yaml` to export the JSON-Schema in both variants.

The output must include the additional attributes as well as an attribute `listName` that includes the `name` of the parameter to allow a user to generate the correct list of parameters. The group should describe a top-level group. 
Example:

```yaml
apiVersion: kudo.dev/v1beta1
parameters:
  - name: NODE_COUNT
    title: "Node Count"
    description: "Number of Cassandra nodes."
    type: number
    default: "3"
  - name: NODE_CPU_MC
    title: "Requested CPU"
    hint: "CPU request (in millicores) for the Cassandra node containers."
    default: "1000"
    type: number
    group: "Node"
    required: true
  - name: NODE_MEM_MIB
    title: "Requested Memory"
    hint: "Memory request (in MiB) for the Cassandra node containers."
    type: number
    default: "4096"
    group: "Node"
    required: true
  - name: NODE_DOCKER_IMAGE
    title: "Docker Image"
    hint: "Cassandra node Docker image"
    description: "The docker image to be used for the Node. Must be one of the mesosphere/cassandra images, as it contains customizations."
    default: "mesosphere/cassandra:3.11.6-1.0.1"
    group: "Node"
    advanced: true
  - name: NODE_DOCKER_IMAGE_PULL_POLICY
    title: "Docker Image Pull Policy"
    hint: "Cassandra node Docker image pull policy"
    description: "The Docker image pull policy. Can be changed, for example if the image is pre-loaded on the Kubernetes nodes."
    default: "Always"
    type: "string"
    enum: 
      - "Always"
      - "Never"
      - "IfNotPresent"
    group: "Node"
    advanced: true
```

The exported JSON-Schema for this parameter definition should look as following:

```json
{
  "title": "cassandra",
  "type": "object",
  "properties": {
    "NODE_COUNT": {
      "type": "number",
      "title": "Node Count",
      "description": "Number of Cassandra nodes.",
      "default": "3"
    },
    "Node": {
      "type": "object",
      "title": "Node",
      "required": [
        "NODE_CPU_MC",
        "NODE_MEM_MIB"
      ],
      "properties":  {
        "NODE_CPU_MC": {
          "listName": "NODE_CPU_MC",
          "type": "number",
          "title": "Requested CPU",
          "hint": "CPU request (in millicores) for the Cassandra node containers.",
          "default": "1000",
          "advanced": false
        },
        "NODE_MEM_MIB": {
          "listName": "NODE_MEM_MIB",
          "type": "number",
          "title": "Requested Memory",
          "hint": "Memory request (in MiB) for the Cassandra node containers.",
          "default": "4096",
          "advanced": false
        },
        "NODE_DOCKER_IMAGE": {
          "listName": "NODE_DOCKER_IMAGE",
          "type": "string",
          "title": "Docker Image",
          "hint": "Cassandra node Docker image",
          "description": "The docker image to be used for the Node. Must be one of the mesosphere/cassandra images, as it contains customizations.",
          "default": "mesosphere/cassandra:3.11.6-1.0.1",
          "advanced": true
        },
        "NODE_DOCKER_IMAGE_PULL_POLICY": {
          "listName": "NODE_DOCKER_IMAGE_PULL_POLICY",
          "type": "string",
          "title": "Docker Image Pull Policy",
          "hint": "Cassandra node Docker image pull policy",
          "description": "The Docker image pull policy. Can be changed, for example if the image is pre-loaded on the Kubernetes nodes.",
          "default": "Always",
          "enum": ["Always", "Never", "IfNotPresent"],
          "advanced": true
        }
      }
    }
  }
}
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

## Alternatives

Fully adopt JSON-Schema, this is described in KEP-33
