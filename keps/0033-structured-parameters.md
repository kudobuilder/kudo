---
kep-number: 33
short-desc: Structured parameters with JSON-schema
title: Structured Parameters
authors:
  - "@zen-dog"
  - "@aneumann82"
owners:
  - "@zen-dog"
  - "@aneumann82"
editor: TBD
creation-date: 2020-08-27
last-updated: 2020-08-31
status: draft
see-also:
replaces:
superseded-by:
---

# Structured Parameters with the JSON-schema

## Table of Contents

<!--ts-->
   * [Structured Parameters with the JSON-schema](#structured-parameters-with-the-json-schema)
      * [Table of Contents](#table-of-contents)
      * [Summary](#summary)
      * [Motivation](#motivation)
         * [Goals](#goals)
         * [Non-Goals](#non-goals)
      * [Proposal](#proposal)
         * [Overview](#overview)
         * [Package Format](#package-format)
         * [CRDs](#crds)
            * [Conversion](#conversion)
            * [Operator](#operator)
            * [OperatorVersion](#operatorversion)
            * [Instance](#instance)
         * [Instance updates](#instance-updates)
         * [Values.yaml](#valuesyaml)
            * [Full Structured values.yaml](#full-structured-valuesyaml)
            * [Partial structured values.yaml](#partial-structured-valuesyaml)
         * [Add Parameter Wizard kudo package add parameter](#add-parameter-wizard-kudo-package-add-parameter)
         * [Parameter Listing kudo package list parameters](#parameter-listing-kudo-package-list-parameters)
         * [Additional attributes in JSON-schema](#additional-attributes-in-json-schema)
            * [Trigger attributes](#trigger-attributes)
            * [Immutability](#immutability)
            * [FlatListName](#flatlistname)
         * [OperatorVersion upgrades](#operatorversion-upgrades)
      * [Example](#example)
         * [Old params.yaml](#old-paramsyaml)
         * [After automatic conversion](#after-automatic-conversion)
         * [Updated OperatorVersion](#updated-operatorversion)
      * [Resources](#resources)

<!-- Added by: aneumann, at: Mon Aug 31 12:21:58 CEST 2020 -->

<!--te-->

## Summary

Replace current flat "list of parameters" with a single, nested and typed structure described by [JSON-schema](https://json-schema.org/).

## Motivation

Currently, KUDO uses a flat [list of parameter](https://github.com/kudobuilder/kudo/blob/main/pkg/apis/kudo/v1beta1/parameter_types.go#L4) which, while being simple and accessible, has its limits like the need to use name prefixes to group the parameters. [JSON-schema](https://json-schema.org) is a de-facto standard when describing a data format. It can also be expressed as YAML to fit into the Kubernetes ecosystem. Along with naming and descriptions it provides following improvements over the existing parameter schema:   
- Labels
- Nested parameters
- Validation and ranged values
- More parameter types (enums, numbers, etc.)
- Easily extendable with custom attributes like `trigger` and `immutable`
- Rich ecosystem of tools and libraries e.g. [UI generators](http://json-schema.org/implementations.html#generators-from-schemas)

Latter point might help to build an ecosystem around KUDO operators which might help driving KUDOs adoption.

### Goals

Introduce a new parameter format based on the JSON-schema.

### Non-Goals

Extend the current format to support new features.

### Proposal Overview

One part of the proposal deals the client-side of the new format. This includes:
 - Introduce new API version (`apiVersion:kudo.dev/v1beta2`) for the operator package `params.yaml`. KUDO should be able to read both old and new formats.
- Support updating nested values from the command line e.g. `kudo update ... -p some.nested.key=value`
- Support updating nested values using values file e.g. `kudo update ... -P values.yaml`
- Add nested parameters via. the wizard `kudo package add parameter`
- Parameter listing aka `kudo package list parameters`. We need a new output format here - maybe a tree-table?
- Possible additional validation for our custom attributes e.g. parameter `trigger`

The server-side of the proposal includes:
- Updating `Operator` and `Instance` CRDs to use JSON-schema as the parameter format  
- Providing a conversion-webhook to convert in-cluster running operators to the new format 
   
### Package Format

JSON-schema is a [well-described](http://json-schema.org/understanding-json-schema/) format. However, here is a little example showing an existing `ZOOKEEPER_URI` and its new counterpart `Zookeeper.URI`:

```yaml
apiVersion: kudo.dev/v1beta1
parameters:
  - name: ZOOKEEPER_URI
    displayName: "zookeeper cluster URI"
    description: "host and port information for Zookeeper connection. e.g. zk:2181,zk2:2181,zk3:2181"
    default: "zookeeper-instance-zookeeper-0.zookeeper-instance-hs:2181"
    required: true
```

```yaml
type: object
"$schema": "http://json-schema.org/draft-07/schema#"
apiVersion: kudo.dev/v1beta2
properties:
  Zookeeper:
    properties:
      URI:
        type: "string"
        description: "host and port information for Zookeeper connection. e.g. zk:2181,zk2:2181,zk3:2181"
        default: "zookeeper-instance-zookeeper-0.zookeeper-instance-hs:2181"
    required: ["URI"]
```

### CRDs

Normally, per Kubernetes conventions, CRDs would evolve over released versions without a breaking change from one version to the next, allowing clients to gradually convert to new CRD versions. This requires for a CRD to be convertible from the old version to new _and vice versa_ for all clients (new and old) to be able to read/write both old and new CRD versions. However, such a conversion in both directions will not be possible with this change, as JSON-schema supports a lot of configurations which will not be mappable from a flat list. 

There will be a hard break from the current from `v1beta1` to the new `v1beta2`. We will provide a migration path for existing installed CRDs, but the clients accessing these CRDs will need to be updated.

#### Conversion

To migrate the existing CRDs, we will implement a Conversion Webhook that only allows `v1beta1` to `v1beta2` conversion, not the other way around. When KUDO is upgraded, it will:

1. Deploy the new CRD version where the new (`v1beta2`) version will be the _only_ served and stored version:
    - `v1beta1` will be still in the list, but with `storage=false, served=false`
    - `v1beta2` will be marked as `storage=true, served=true`
2. Deploy the new manager with the conversion webhook
3. Run Pre-Migration: fetch all existing CRs, run the conversion and check for any errors
4. Run the migration: fetch all CRDs and issue an update - This will trigger a conversion and save the CRD with the new version
5. With a later major KUDO release, we can remove `v1beta1` from the CRD list - this should be at least 6 month later, as soon as we remove support for the `v1beta1` version we can not support migrating existing installations anymore.

Reference: [CRD Versioning](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/)

The actual type that we are going to use in the CRD is undefined yet - The CRD generation fails to accept untyped `interface{}`. There is an option
to use raw `[]byte` data, or we might be able to define a typed structure to map a JSON schema into.
 
#### Operator

The Operator CRD will not change, but we should keep the version in sync with the other CRDs.

#### OperatorVersion

OperatorVersion `v1beta1` [holds the parameters](https://github.com/kudobuilder/kudo/blob/main/pkg/apis/kudo/v1beta1/operatorversion_types.go#L35) in the `OperatorVersion.Spec.Parameter` field of type `[]Parameter`. `v1beta2` will update this field to hold a JSON-schema instead. This structure will be mapped in the CRD to an `interface{}` type and will be read and interpreted by a go library to e.g. validate input parameters and provide parameter defaults.

`v1beta1`:
```go
type OperatorVersionSpec struct {
    ....
    Parameters []Parameter `json:"parameters,omitempty"`
    ....
}
```

`v1beta2`:
```go
type OperatorVersionSpec struct {
    ....
    Parameters interface{} `json:"parameters"`
    ....
}
```

#### Instance

Instance `v1beta1` saves parameter overrides in the `Instance.Spec.Parameters` field which is of type `map[string]string`. `v1beta2` will have to update this field to `interface{}`.

`v1beta1`:
```go
type InstanceSpec struct {
    ....
	Parameters map[string]string `json:"parameters,omitempty"`
    ....
}
```

`v1beta2`:
```go
type InstanceSpec struct {
    ....
    Parameters map[string]interface{} `json:"parameters"`
    ....
}
```

The existing instance parameter values are currently stored as serialized strings. They will have to be unwrapped and stored as `map[string]interface{}`.

### Instance updates

When updating an operator instance, a user can specify parameters override either in the command line directly (`-p` option) or using an extra file (`-P` option) e.g.: `kudo update --instance my-instance -p KEY=newValue`. This is easy with a flat list of parameters, but becomes more complex with a nested structure. 

The key is now be a [JSON pointer](https://tools.ietf.org/html/rfc6901) expression, the value can either be a simple value, or a more complex json/yaml structure, depending on the specified path. On an update, the existing parameter structure is modified based on the given parameter overrides and only after all updates are applied it is validated against the JSON-schema of the operator.

Examples:
- set the top-level `clusterName` parameter: 
  `kudo update ... -p clusterName="NewClusterName"`
- set the `datacenterName` field of the first item in the `topology` array
  `kudo update ... -p topology/0/datacenterName="NewDC1"`
- replace the first item in the `labels` array of the first `topology` array element
  `kudo update ... -p topology/0/labels/0={ key: "Usage", value: "newValue" }`
- replace the full labels array of the first topology element
`kudo update ... -p topology/0/labels=[ { key: "DCLabel", value: "dc1" }, { key: "Usage", value: "additionalItem" } ]`

The `kudo update` command will not allow adding or removing a single entry from an array field, as this would make the command not idempotent anymore.

We may decide to implement a `kudo patch` later on which is not idempotent an allows add/remove operations.

### Values.yaml

When installing or updating an operator, users can use a file that contains parameter values: `kudo update --instance my-instance -P values.yaml`

There are three different variants:
 - Full structure (and replace)
 - Partial structure (and merge)

#### Full Structured values.yaml

In this variant, the values.yaml will always contain the full parameter structure. It would replace the whole parameter structure. This would be a change from the current behaviour, where the `values.yaml` contains a list of parameters that is merged with the existing list in an instance.

```yaml
clusterName: my-cluster
topology:
- datacenterName: DC1
  nodeCount: 3
  labels:
  - key: DCLabel
    value: datacenter1
- datacenterName: DC2
  nodeCount: 5
  labels:
  - key: DCLabel
    value: datacenter2
backup:
  enabled: true
  name: "MyBackup"
  credentials:
    name: my-username
    password: my-pass
```

#### Partial structured values.yaml

Similar to the full structure, but we merge the given structure with the existing parameter structure. Object types are merged, arrays are fully replaced.

```yaml
clusterName: my-cluster
backup:
  credentials:
    name: new-username
```


### Add Parameter Wizard `kudo package add parameter`
    
TODO: We need a new/extended implementation here


### Parameter Listing `kudo package list parameters`

TODO: We need a new output format here - maybe a tree table

### Additional attributes in JSON-schema

 - `trigger`: Copied from params.yaml, describes the plan that is triggered
 - `immutable`: Copied from params.yaml, specifies if the field is updatable (Could this clash with a future addition to JSON-schema? Should we rename it?)
 - `flatListName`: Provided for backwards compatibility. If provided, specifies the old name that was used in the flat parameters list in previous versions (alternatives: "oldName", "paramName")? This attribute should be deprecated at some point in the future and phased out.
- `low-priority`: Optional new boolean parameter that specifies that this parameter is not normally expected to be changed. Very advanced use cases might require a change, but it's usually not required.

#### Trigger attributes

With the list of parameters, each parameter had a trigger attribute which specified a plan to trigger when the parameter
was changed.

With the nested structure, we need to define where and how a plan can be triggered:
 - Every level of the structure can specify a trigger
 - Fields can explicitley specify an empty trigger field. This means to *not* trigger a plan.
 - The root is allowed to *not* have a trigger
 - If a field changes, KUDO traverses the structure upwards until it finds a field with a trigger or reaches the root.
   If no trigger is found, no plan is triggered.

The previously established rule applies:
 - If more than one plan is triggered in a single update, the update is not allowed.

#### Immutability

If a field is marked as immutable, this applies to the field itself as well as all potential children.

#### FlatListName

This attribute is important for operator developers to provide backwards compatibility. Existing installed versions of an operator contain the parameters as a list, and with the `flatListName` KUDO can automatically convert an  instance with a flat list of parameters to a new nested structure.

### OperatorVersion upgrades

Until now, an operator version upgrade did not require any special handling from KUDO. There is a special `upgrade` plan that is triggered in this instance, but apart from that no special code path.

With the structured parameters there is a potential migration

## Example

This section provides an example of conversion

### Old `params.yaml`

This is the old `params.yaml` from an operator

```yaml
apiVersion: kudo.dev/v1beta1
parameters:
  - name: CLUSTER_NAME
    default: "my-cluster"
    required: true
  - name: BACKUP_ENABLED
    default: false
  - name: BACKUP_CREDENTIALS_USERNAME
    displayName: "AWS Credentials Username"
  - name: BACKUP_CREDENTIALS_PASSWORD
    displayName: "AWS Credentials Password"
```

The templates in the operator use `{{ .Params.CLUSTER_NAME }}` and `{{ .Params.BACKUP_CREDENTIALS_USERNAME }}` to insert the parameters.

### After automatic conversion

KUDO will automatically convert this params.yaml into a JSON-schema inside the operator version:

```yaml
title: Parameter Schema
description: TODO
type: object
required:
- CLUSTER_NAME
properties:
  CLUSTER_NAME:
    type: string
    default: my-cluster
  BACKUP_ENABLED:
    type: boolean
    default: false
  BACKUP_CREDENTIALS_USERNAME:
    type: "string"
    title: "AWS Credentials Username"
  BACKUP_CREDENTIALS_PASSWORD:
    type: "string"
    title: "AWS Credentials Passsword"    
```

This converted schema results in a parameter structure like this (depending on the values the user provided):

```yaml
CLUSTER_NAME: "my-cluster"
BACKUP_ENABLED: false
BACKUP_CREDENTIALS_USERNAME: "some value"
BACKUP_CREDENTIALS_PASSWORD: "some password"
```

The templates in the operator contintue to use `{{ .Params.CLUSTER_NAME }}` and `{{ .Params.BACKUP_CREDENTIALS_USERNAME }}` to insert the parameters, no change to Operators is required.


### Updated OperatorVersion

After a while, the operator developer updates the operator to make use of the structured parameters and provides new templates and a `json-schema.yaml`:

```yaml
title: My Operator
description: All parameters
type: object
required:
- clusterName
properties:
  clusterName:
    type: string
    title: ClusterName
    description: "The name of the cluster to create"
    default: my-cluster
    flatListName: CLUSTER_NAME
  backup:
    type: object
    title: Backup
    description: "Configuration for cluster Backup"
    required:
    - enabled    
    properties:
      enabled:
        type: boolean
        flatListName: BACKUP_ENABLED
      credentials:
        type: object
        title: AWS Credentials
        required:
        - name
        - password        
        properties:
          name:
            type: string
            title: Username
            flatListName: BACKUP_CREDENTIALS_USERNAME
          password:
            type: string
            title: Password
            flatListName: BACKUP_CREDENTIALS_PASSWORD
```

The templates are updated and use the nested structure now: `{{ .Params.clusterName }}` and `{{ .Params.backup.credentials.name }}` 

As the operator developer provided the `flatListName` attributes in the schema, KUDO can automatically update existing operator instances and convert the parameter list into a structure:

```yaml
clusterName: "my-cluster"
backup:
  enabled: false
  credentials:
    name: "some value"
    password: "some password"
```

TODO: Extend when exactly this should happen and in which part of the code

## Resources

- https://json-schema.org/understanding-json-schema/index.html
- https://rjsf-team.github.io/react-jsonschema-form/
