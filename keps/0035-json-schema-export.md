---
kep-number: 35
short-desc: JSON-schema export
title: JSON-schema export
authors:
  - "@aneumann82"
owners:
  - "@aneumann82"
creation-date: 2020-09-22
last-updated: 2020-09-29
status: provisional
see-also:
  - KEP-33
---

# JSON-schema export

## Table of Contents

<!--ts-->
   * [JSON-schema export](#json-schema-export)
      * [Table of Contents](#table-of-contents)
      * [Summary](#summary)
      * [Motivation](#motivation)
         * [Goals](#goals)
         * [Non-Goals](#non-goals)
      * [Proposal](#proposal)
         * [User Stories](#user-stories)
            * [Story 1](#story-1)
         * [Implementation Details](#implementation-details)
            * [Additional Types:](#additional-types)
            * [Additional Attributes](#additional-attributes)
            * [Groups](#groups)
         * [JSON-Schema export](#json-schema-export-1)
      * [Alternatives](#alternatives)
      * [Implementation History](#implementation-history)

<!-- Added by: aneumann, at: Tue Sep 29 15:37:14 CEST 2020 -->

<!--te-->

## Summary

Extend the existing `params.yaml` with metadata and new types and add an export as JSON-schema.

## Motivation

JSON-schema is the preferred way to generate a user-friendly UI to configure a new instance of a KUDO operator or to trigger plans with parameters. KUDO should be able to export a JSON-schema that describes the required parameters.

KEP-33 proposes a full replacement of the existing flat list `params.yaml` with a nested and typed structure described by JSON-schema. This is a big breaking change and requires a lot of work to get right in a single release.

This KEP proposes to enhance the flat list parameters with additional attributes and add support to export a JSON-schema. This is a first step towards KEP-33, it allows tooling to be build based on JSON-Schema while not introducing breaking changes on the operator level.

### Goals

- Add additional attributes required for a useful JSON-schema and UI generation
- Add support to export a JSON-schema that describes the parameters
- Prepare Implementation of KEP-33 in the future 

### Non-Goals

- Handle multiple CRD versions
- Replace the `params.yaml` with a json-schema
- Add validation for parameters

## Proposal

Add new types as well as optional new attributes to enhance the parameters and bring them closer to JSON-Schema.

Implement an export that generates a valid JSON-Schema out of the flat parameter list.

### User Stories

#### Story 1

A user wants to generate a UI to install a KUDO operator. He wants to use an automatic generator, for example https://rjsf-team.github.io/react-jsonschema-form/

### Implementation Details

#### Additional Types: 

To bring the parameters up to par with JSON-schema, we add these additional types:
- `boolean` A "true" or "false" value, from the JSON "true" or "false" productions
- `number` An arbitrary-precision, base-10 decimal number value, from the JSON "number" production

#### Additional Attributes

JSON-Schema already provides good support for generated UIs, but it can be enhanced. Additionally, we do not want to adopt the full nested structure with alternatives and references yet, there is a different KEP for that. The additional attributes should provide a step in the direction to generate a good UI from it

All of the additional attributes are fully optional.

`group` 
An attribute of type "string". This attribute describes a top-level group used for this parameter.

`advanced` 
An attribute of type "boolean" that indicates that this parameter is for advanced configuration and can be hidden by default. Defaults to "false". Parameters that have "advanced=true" *must* have a default value, as they might not be visible immediately.

`hint`
An attribute of type "string" that should provide a short description about the syntax of this parameter. Usually shown below the actual input element.

`enum` 
An array of values that are allowed to be used. Each entry must be of the type defined by the `type` attribute. If a default is specified, it must be listed as well.


Example:

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

#### Groups

Add a top-level element `groups` that describes the groups used:

```
apiVersion: kudo.dev/v1beta1
groups:
  - name: "Node"
    title: "Node Configuration"
    description: "This section contains parameters that are related to the Node"
parameters:
...
```

This `groups` element is fully optional and does not affect any behaviour of KUDO. We may decide to not add this to the OperatorVersion CRD but to keep it only in the `params.yaml` package format, that is an implementation detail. The data provided in this element can be used by the JSON-Schema export to fill in the metadata (title and description) of top level groups.

### JSON-Schema export

Extend the `kudo package list parameters` with an option `--output json-schema` to describe the parameters as JSON-Schema.

Alternatively, add a new command `kudo package list json-schema`, which then could be used with `--output json` or `--output yaml` to export the JSON-Schema in both variants.

The output must include the additional attributes as well as an attribute `listName` that includes the `name` of the parameter to allow a user to generate the correct list of parameters. The group should describe a top-level group. 
Example:

```yaml
apiVersion: kudo.dev/v1beta1
groups:
  - name: "Node"
    title: "Node Configuration"
    description: "This section contains parameters that are related to the Node"
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
      "title": "Node Configuration",
      "description": "This section contains parameters that are related to the Node",
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

## Alternatives

Fully adopt JSON-Schema, this is described in [KEP-33](0033-structured-parameters.md)

## Implementation History

- 2020-09-22 - Initial draft (@aneumann82)
- 2020-09-29 - Updated KEP with examples and enum (@aneumann82)
