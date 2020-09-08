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
status: provisional
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
         * [Proposal Overview](#proposal-overview)
         * [Package Format](#package-format)
         * [CRDs](#crds)
            * [Conversion](#conversion)
            * [Operator](#operator)
            * [OperatorVersion](#operatorversion)
            * [Instance](#instance)
         * [Client-side](#client-side)
            * [Instance updates from the command line (-p option)](#instance-updates-from-the-command-line--p-option)
            * [Instance updates with a parameter file (-P option)](#instance-updates-with-a-parameter-file--p-option)
            * [Add Parameter Wizard kudo package add parameter](#add-parameter-wizard-kudo-package-add-parameter)
            * [Parameter Listing kudo package list parameters](#parameter-listing-kudo-package-list-parameters)
      * [Additional attributes in JSON-schema](#additional-attributes-in-json-schema)
         * [Trigger](#trigger)
         * [Immutable](#immutable)
         * [OldName](#oldname)
      * [Operator conversion and upgrade example](#operator-conversion-and-upgrade-example)
         * [kudo.dev/v1beta1 operator version](#kudodevv1beta1-operator-version)
         * [Conversion to kudo.dev/v1beta2](#conversion-to-kudodevv1beta2)
         * [Upgrading operator](#upgrading-operator)
      * [Resources](#resources)

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

This latter point might help to build an ecosystem around KUDO operators which might help driving KUDOs adoption.

### Goals

Introduce a new parameter format based on the JSON-schema.

### Non-Goals

Extend the current format to support new features.

### Proposal Overview

One part of the proposal deals the client-side of the new format. This includes:
 - Introduce new API version (`apiVersion:kudo.dev/v1beta2`) for the operator package `params.yaml`. KUDO should be able to read both old and new formats.
- Support updating nested values from the command line e.g. `kudo update ... -p some.nested.key=value`
- Support updating nested values using values file e.g. `kudo update ... -P values.yaml`
- Add nested parameters via the wizard `kudo package add parameter`
- Parameter listing aka `kudo package list parameters`. We may need to consider a new output format here - maybe a tree-table?
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

KUDO will support both package formats for the future. At some point the old format might become deprecated, but this is not planned yet.

Note: The apiVersions here (`v1beta1` and `v2beta2`) correspond to the CRD versions, but are separate definitions, even if they seem to be similar in certain aspects.

### CRDs

Normally, per Kubernetes conventions, CRDs would evolve over released versions without a breaking change from one version to the next, allowing clients to gradually convert to new CRD versions. This requires for a CRD to be convertible from the old version to new _and vice versa_ for all clients (new and old) to be able to read/write both old and new CRD versions. However, such a conversion in both directions will not be possible with this change, as JSON-schema supports a lot of configurations which will not be mappable from a flat list. 

There will be a hard break for clients and users of CRDs from `v1beta1` to the new `v1beta2`. We will provide a migration path for existing installed CRDs, but the clients accessing these CRDs will need to be updated.

#### Conversion

To migrate the existing CRDs, we will implement a Conversion Webhook that only allows `v1beta1` to `v1beta2` conversion, not the other way around. When KUDO is upgraded, it will:

1. Run Pre-Migration: fetch all existing CRs in version `v1beta1`, run the conversion to `v1beta2` and check for any errors. If this fails, the upgrade will be aborted and the user need to manually investigate why the conversion failed.
2. Deploy the new CRD version where the new (`v1beta2`) version will be the _only_ served and stored version:
    - `v1beta1` will be still in the list, but with `storage=false, served=false`
    - `v1beta2` will be marked as `storage=true, served=true`
3. Deploy the new manager with the conversion webhook. This will start the reconciliation loop again and KUDO will start working.
4. Run the migration: fetch all CRDs and issue an update - This will trigger a conversion and save the CRD with the new version. This is not neccessarily required, but it will make sure that all existing CRs are saved as `v1beta2`. When the migration has finished, we can remove the `v1beta1` from the CRDs `status.storedVersions` list to indicate that no more old versions are stored in the etcd.
5. With a later major KUDO release, we can remove `v1beta1` from the CRD list - this should be at least 6 month later, as soon as we remove support for the `v1beta1` version we cannot support migrating existing installations anymore.

The conversion webhook is stable in Kubernetes 1.16, and beta (and therefore enabled by default) in Kubernetes 1.15. 

Reference: [CRD Versioning](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/)

#### Operator

The Operator CRD will not change, but we should keep the version in sync with the other CRDs.

#### OperatorVersion

OperatorVersion `v1beta1` [holds the parameters](https://github.com/kudobuilder/kudo/blob/main/pkg/apis/kudo/v1beta1/operatorversion_types.go#L35) in the `OperatorVersion.Spec.Parameter` field of type `[]Parameter`. `v1beta2` will update this field to hold a JSON-schema instead. This structure will be mapped in the CRD to a raw `extensionapi.JSON` and will be read and interpreted by a go library to e.g. validate input parameters and provide parameter defaults.

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
    Parameters extensionapi.JSON `json:"parameters"`
    ....
}
```

A better solution than `extensionapi.JSON` would be to have a typed structure, but CRDs currently cannot describe recursive data structures, as they [don't support $ref or $recursiveRef](https://github.com/kubernetes/kubernetes/issues/54579). The only way to define an arbitrary JSON structure in a Kubernetes API is with `apiextension.JSON`. The resulting CRD for the operator version will therefore look like this:

```yaml
...
              parameters:
                description: The parameter definition. Must be a valid json-schema in version TODO. See https://json-schema.org/
                x-kubernetes-preserve-unknown-fields: true
...
```

Here is an example: give the `v1beta1` operator version parameters defined as:

```yaml
spec:
  parameters:
  - name: NODE_COUNT
    description: Number of replicas that should be run as part of the deployment
    default: 3
    required: true
  - name: BACKUP_ENABLED
    default: false
```

it will be converted into `v1beta2` and will look like:

```yaml
title: Parameter Schema
"$schema": "https://json-schema.org/draft/2019-09/schema"
type: object
required:
- NODE_COUNT
properties:
  NODE_COUNT:
    type: integer
    default: 3
    description: "Number of replicas that should be run as part of the deployment"
  BACKUP_ENABLED:
    type: boolean
    default: false
```

Note, that the parameter named haven't changed, and they all live in the root-level object. This way, they still can be referenced in the templates as e.g. `{{ .Params.NODE_COUNT }}`

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
    Parameters apiextension.JSON `json:"parameters"`
    ....
}
```

The existing instance parameter values are currently stored as serialized strings. They will have to be unwrapped and stored as `apiextension.JSON`. The Instance should have accessor or helper functions to convert the `apiextension.JSON` values to their actual types using the JSON-schema provided by the corresponding `OperatorVersion`. 

Here is an example: give a `v1beta1` instance parameters like:

```yaml
spec:
  parameters:
    NODE_COUNT: "3"
    BACKUP_ENABLED: "false"
    ARRAY_PARAM: '[foo,bar,bazz]'
    MAP_PARAM: '{foo:"bar"}'
```

they will be converted into `v1beta2` and will look like:

```yaml
spec:
  parameters:
    NODE_COUNT: 3
    BACKUP_ENABLED: false
    ARRAY_PARAM:
    - foo
    - bar
    - bazz
    MAP_PARAM: 
      foo: "bar"
```

// TODO: it seems that the conversion webhook doesn't have access to the cluster resources (at least by default). How do we unwrap the values (e.g. `MAP_PARAM`) without access to the `OperatorVersion` JSON-schema?

### Client-side

#### Instance updates from the command line (`-p` option)

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

The `kudo update` command will not allow adding or removing a single entry from an array field, as this would make the command not idempotent anymore. The only way to add or remove an entry from an array field is to replace the full array.

We may decide to implement a `kudo patch` later on which is not idempotent an allows add/remove operations. For example:

- Add a new label entry to the first topology array element
  `kudo patch ... --add topology/0/labels={ key: "NewKey", value: "NewValue" }`
- Remove the second label entry of the first topology array element
  `kudo patch ... --remove topology/0/labels/1`

The implementation of a `patch` command is *not* part of this KEP, these examples are to show possible options.

#### Instance updates with a parameter file (`-P` option)

When installing or updating an operator, users can use a file that contains parameter values: `kudo update --instance my-instance -P values.yaml`. We will merge the existing with the updated parameter using [JSON merge patch](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/#use-a-json-merge-patch-to-update-a-deployment). This means, that to e.g. specifying a new array element, the whole array has to be provided. 

#### Add Parameter Wizard `kudo package add parameter`

The `add parameter` wizard needs to be rewritten and adjusted to the new format. In addition to the current data, it must ask the user for the data type of the new parameter and A JSON pointer in the existing structure. This, however, only applies to simple schemas as any advanced functionality like reusing existing schema definitions, schema combinations (`anyOf`, `oneOf` etc) are hard to express in the CLI.

 It is unclear how practical generating a JSON-schema using a CLI wizard is. There is, however, a number of tools that can [generate a schema](https://www.liquid-technologies.com/online-json-to-schema-converter) given a valid JSON document.

#### Parameter Listing `kudo package list parameters`

The output for the parameter list must be adjusted. The first column of the `uitable` should contain an indented structure that allows a user to see the nested nature of the parameters.

## Additional attributes in JSON-schema

 - `trigger` Copied from params.yaml, describes the plan that is triggered
 - `immutable` Copied from params.yaml, specifies if the field is updatable (Could this clash with a future addition to JSON-schema? Should we rename it?)
 - `oldName` Provided for backwards compatibility. If provided, specifies the old name that was used in the flat parameters list in the previous version? Having the ability to map a parameter from the previous version to the current one, gives the operator developers the ability to update their schemas and rearrange/rename the parameters.

### Trigger

With the list of parameters, each parameter had a trigger attribute which specified a plan to trigger when the parameter was changed. With the nested structure, this gets more complicated as one "parameter branch" might have multiple different triggers defined which is clearly not something we want. Here is the new rule set:
 - Every level of the structure *may* specify a trigger.
 - If a field does not explicitly specify a trigger, the trigger is inherited from the parent.
 - A deeper level of the structure *must not* specify a trigger if any parent specifies a trigger.
 - If a field changes, KUDO traverses the structure upwards until it finds a field with a trigger or reaches the root. If it does not find a trigger the `deploy` plan is triggered, similar to the old behaviour.

The previously established rule applies:
 - If more than one plan is triggered in a single update, the update is not allowed.

**Examples:**

Invalid: A child object `Authorization.Enabled` level triggers a different plan than its parent `Authorization`
```yaml
type: object
properties:
  Authorization:
    type: object
    trigger: "deploy" # <-- triggering "deploy"
    properties:
      ENABLED:
        type: "boolean"
        trigger: "update-instance" # <-- triggering "update-instance"
```

Valid: The following rules apply:
- A change to `ClusterName` will trigger `deploy` (directly specified)
- A change to `NodeCount` will trigger `deploy`. There is no specified trigger, so we look at the parent. Parent doesn't have a trigger too, so `deploy` is triggered by default.
- A change to `Authorization.ENABLED` will trigger `auth-update` (directly specified.)
- A change to `Authorization.NAME` will trigger `auth-update` (parent trigger is applied)

```yaml
type: object
properties:
  ClusterName:
    type: string
    trigger: "deploy"
  NodeCount: 
    type: int
  Authorization:
    type: object
    trigger: "auth-update" # <-- triggering "auth-update"
    properties:
      ENABLED:
        type: "boolean"
        trigger: "auth-update"
      NAME:
        type: "string"
```

### Immutable

If a field is marked as `immutable`, this applies to the field itself as well as all children.

### OldName

This attribute is important for operator developers to provide backwards compatibility. After the conversion from `v1beta1` to `v1beta2` an operator developer might want to release a new operator version which fully utilises the JSON-schema possibilities and re-arranges existing parameters. Here,  `oldName` is used to map old parameters to the new ones. Let's walk through a complex example, that takes an already installed `v1beta1` operator, converts it to `v1beta2` and upgrades it to a new operator version, which updates and renames the parameters using the `oldName` attribute. 

## Operator conversion and upgrade example

### kudo.dev/v1beta1 operator version

Let's assume we have an existing `v1beta1` `OperatorVersion` with parameters looking like this: 

```yaml
apiVersion: kudo.dev/v1beta1
kind: OperatorVersion
spec:
  parameters:
    - name: CLUSTER_NAME
      default: "my-cluster"
      required: true
    - name: BACKUP_ENABLED
      default: false
    - name: BACKUP_CREDENTIALS_USERNAME
      description: "AWS Credentials Username"
    - name: BACKUP_CREDENTIALS_PASSWORD
      description: "AWS Credentials Password"
```

this flat parameter list practically mirrors operator's `params.yaml` file. The templates in the operator use e.g. `{{ .Params.CLUSTER_NAME }}` and `{{ .Params.BACKUP_CREDENTIALS_USERNAME }}` to reference the parameters.

### Conversion to kudo.dev/v1beta2

When updating to the new KUDO version, KUDO migration process will install the conversion webhook and trigger the conversion on all existing `OperatorVersion` resources (no changes in the `Operator` are required). Note, that the same conversion happens when an old Operator with `v1beta1` `params.yaml` is installed on a cluster with a new `v1beta2` KUDO version.

```yaml
apiVersion: kudo.dev/v1beta1
kind: OperatorVersion
spec:
  parameters:
    title: Parameter Schema
    "$schema": "https://json-schema.org/draft/2019-09/schema"
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
        description: "AWS Credentials Username"
      BACKUP_CREDENTIALS_PASSWORD:
        type: "string"
        description: "AWS Credentials Password"   
```

The operator `Instance` spec parameters stay unchanged (provided the user set all four parameters):

```yaml
apiVersion: kudo.dev/v1beta1
kind: Instance
spec:
  parameters:
    CLUSTER_NAME: "my-cluster"
    BACKUP_ENABLED: false
    BACKUP_CREDENTIALS_USERNAME: "some value"
    BACKUP_CREDENTIALS_PASSWORD: "some password"
```

The templates in the operator continue to use `{{ .Params.CLUSTER_NAME }}` and `{{ .Params.BACKUP_CREDENTIALS_USERNAME }}`.

### Upgrading operator

After a while, the operator developer updates the operator to make use of the structured parameters. The parameters are restructured and renamed, templates are updated. The new parameters schema look like following:

```yaml
title: My Operator
"$schema": "https://json-schema.org/draft/2019-09/schema"
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
    oldName: CLUSTER_NAME
  backup:
    type: object
    title: Backup
    description: "Configuration for cluster Backup"
    required:
    - enabled    
    properties:
      enabled:
        type: boolean
        oldName: BACKUP_ENABLED
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
            oldName: BACKUP_CREDENTIALS_USERNAME
          password:
            type: string
            title: Password
            oldName: BACKUP_CREDENTIALS_PASSWORD
```

The templates are updated and now use the nested structure: `{{ .Params.clusterName }}` and `{{ .Params.backup.credentials.name }}`. As the operator developer provided the `oldName` attributes in the schema, KUDO can automatically update existing operator instances and convert the parameter map into a structure:

Old instance parameters:

```yaml
CLUSTER_NAME: "my-cluster"
BACKUP_ENABLED: false
BACKUP_CREDENTIALS_USERNAME: "some value"
BACKUP_CREDENTIALS_PASSWORD: "some password"
```

are converted into:

```yaml
clusterName: "my-cluster"
backup:
  enabled: false
  credentials:
    name: "some value"
    password: "some password"
```

The conversion happens in the instance admission webhook when the upgrade is detected. KUDO traverses the parameter definition and for every encountered `oldName` looks up the referenced parameter value in the old `Instance` to build the new parameter value structure.

While being particularly useful when migrating from old (and flat) parameter into the new (and structured) ones, `oldName` can also be used later by the operator developer to restructure existing parameter schema so that once `CLUSTER_NAME` (operatorVersion 0.1) can be converted into `clusterName` (operatorVersion 0.2) and later moved into `Cluster.Name` (operatorVersion 0.3). This, however, introduces another challenge where some operator versions (e.g. 0.2 in our example) become "none-skippable" as otherwise the `oldName` will not be found. 

## Resources

- https://json-schema.org/understanding-json-schema/index.html
- https://rjsf-team.github.io/react-jsonschema-form/
