---
kep-number: 8
title: Operator Testing
short-desc: Description of how to test operators
authors:
  - "@jbarrick-mesosphere"
owners:
  - "TBD"
editor: TBD
creation-date: 2019-05-03
last-updated: 2019-05-03
status: implementable
see-also:
  - KEP-0004
---

# Operator Testing

## Table of Contents

  * [Summary](#summary)
  * [Motivation](#motivation)
     * [Goals](#goals)
     * [Non-Goals](#non-goals)
  * [User Stories](#user-stories)
     * [KUDO Developers](#kudo-developers)
     * [Operator Developer](#operator-developer)
     * [Cluster Administrator](#cluster-administrator)
     * [Application Operator](#application-operator)
  * [Risks and Mitigations](#risks-and-mitigations)
  * [Proposal](#proposal)
     * [Definitions](#definitions)
     * [Running the tests](#running-the-tests)
     * [Directory structure](#directory-structure)
     * [Test step file structure](#test-step-file-structure)
        * [TestStep object](#teststep-object)
        * [Assertion files](#assertion-files)
          * [Resources that have non-deterministic names](#resources-that-have-non-deterministic-names)
          * [TestAssert object](#testassert-object)
        * [Test constraints](#test-constraints)
     * [Results collection](#results-collection)
     * [Garbage collection](#garbage-collection)
  * [Graduation Criteria](#graduation-criteria)
  * [Implementation History](#implementation-history)
  * [Alternatives](#alternatives)
  * [Infrastructure Needed](#infrastructure-needed)

## Summary

In order to ensure the reliability of Operators built on KUDO, it is important that developers of Operators are able to simply author and run tests that validate that their Operators are running correctly. Tests can be run in CI to verify changes, by Operator developers in their development cycle, and by cluster administrators to verify that Operators are fully functional in their Kubernetes clusters, across a variety of configurations.

This document outlines a simple, declarative test harness for authoring and running acceptance test suites for KUDO Operators.

## Motivation

### Goals

* Allow Operator developers to write acceptance and unit test cases for Operators.
* Allow Application Operators and Cluster Administrators to write test cases that validate their customizations to Operators.
* Run test suites for Operators on pull requests for Operators.
* Support running test suites via a `kubectl kudo` subcommand against any cluster.
* Integration with Sonobuoy to validate Operators in any cluster.
* Ensure that KUDO Operator developers have a consistent development experience - from writing their operators to testing them.

### Non-Goals

* Outlining testing policies for Operators (e.g., minimum coverage to graduate to "stable").
* Provisioning Kubernetes test infrastructure.

## User Stories

#### KUDO Developers

As the KUDO development team, I...

* *Must* be able to validate that KUDO releases do not break existing Operators.
* *Must* be able to validate that Operator contributions do not break existing Operators.
* *Must* be able to validate Operators in many different Kubernetes environments to ensure compatibility.

#### Operator Developer

As an Operator Developer, I...

* *Must* be able to validate that my changes do not break my Operators and that my features work correctly.
* *Must* be able to write test cases for many different valid (and invalid) configurations.
* *Want to* be able to author tests in a declarative language, without resorting to writing code in a normal programming language.
* *Must* be able to easily understand test results and failures.
* *Must* be able to incorporate KUDO Operator plans into my tests.

#### Cluster Administrator

As a Cluster Administrator, I...

* *Must* be able to easily validate that Operators work well in my cluster.
* *Want to* be able to author my own test cases for known failure modes or critical features.

#### Application Operator

As an Application Operator, I...

* *Must* be able to validate that my Instances are working correctly.

## Risks and Mitigations

* Perhaps the most important consideration when building any testing pipeline is ensuring that developer velocity is not impacted. This is especially important for KUDO, since the developer experience for Operator developers is critical to the success of the project. To mitigate this, we need to optimize the runtime of the test suites and pull request flow. Operator contributions should only run tests for the Operators that are affected by the pull request. Tests should have a fixed upper time limit to prevent any new tests from causing excessive delays. Tests fail early.
* If the test harness is not designed well, contributions to the project become very difficult or the harness does not get adopted by developers. We provide a minimal, declarative format for authoring tests to encourage adoption and provide simple ways to execute the tests to ensure that it is easy to get started.
* The test harness is not valuable if no test cases are written. This means Operator developers need to write test cases! We encourage test case authoring by writing good documentation, integrating the test harness into the CI/CD pipeline, and potentially enacting policies around testing for official Operators. We can also advocate for testing via blogs and interactions with Operator developers.
* As we take a declarative approach to testing, it's important to ensure that the abstractions built are not leaky. Leaky abstractions lead to greater complexity and make the test suite hard to use. We decrease this risk by narrowing the scope to focus on the state of Kubernetes objects in the Kubernetes API, refering to prior work (see [Alternatives](#Alternatives)), and building a minimal solution. We also will write tests for existing Operators and work with authors of new Operators to ensure that the solution is generally useful.

## Proposal

Test cases will be authored by defining Kubernetes objects to apply and Kubernetes state to wait for. Each test case will run in its own namespace (test cases can run concurrently), within each test case, test steps run sequentially. This allows test authors to model state transitions of Operators as different configurations are applied.

### Definitions


* `Operator`: A KUDO Operator.
* `Test Harness`: the tool that runs a test suite.
* `Test Suite`: a collection of test cases.
* `Test Case`: a single, self contained test - can be run in parallel to other test cases.
* `Test Step`: a portion of a test case indicating a state to apply and expect, dependent on all previous test steps in the test case being successful.
* `Assertion`: the expected state of a test step.

### Running the tests

Tests will be invoked via a CLI tool that runs the test suite against the current Kubernetes context by default, or optionally against a mocked control plane or kind (kubernetes-in-docker) cluster. It will be packaged with the default set of test suites from the KUDO Operators repository using go-bindata, with the ability to provide alternative directories containing tests to use.

The tool will enumerate each test case (group of test steps) and run them concurrently in batches. Each test case will run in its own namespace (care must be taken that cluster-level resources do not collide with other test cases [??? TODO: solvable?]) which will be deleted after the test case has been completed. Resources in test steps that are namespace-level will have their namespace set to the test case namespace if it is not present.

Each test case consists of a directory containing test steps and their expected results ("assertions"). The test steps within a test case are run sequentially, waiting for the assertions to be true, a timeout to pass, or a failure condition to be met.

Once all of the test steps in a test case have run, a report for the test case is generated and all of its resources are deleted unless the harness is configured not to clean up resources.

The test harness can also be run in a unit testing mode, where it spins up a dummy Kubernetes API server and runs any test cases marked as unit tests.

### Test suite configuration

The test suite is configured either using command line arguments to `kubectl kudo test` or a provided configuration file.

The configuration format is a YAML file containing the following struct:

```
type TestSuite struct {
	TypeMeta
	ObjectMeta

	// Path to CRDs to install before running tests.
	CRDDir            string
	// Paths to directories containing manifests to install before running tests.
	ManifestDirs      []string
	// Directories containing test cases to run.
	TestDirs          []string
	// Kubectl specifies a list of kubectl commands to run prior to running the tests.
	Kubectl []string `json:"kubectl"`
	// Commands to run prior to running the tests. 
	Commands []Command `json:"commands"`
	// Whether or not to start a local etcd and kubernetes API server for the tests (cannot be set with StartKIND)
	StartControlPlane bool
	// Whether or not to start a local kind cluster for the tests (cannot be set with StartControlPlane).
	StartKIND bool `json:"startKIND"`
	// Path to the KIND configuration file to use (implies StartKiIND).
	KINDConfig string `json:"kindConfig"`
	// KIND context to use.
	KINDContext string `json:"kindContext"`
	// If set, each node defined in the kind configuration will have a docker named volume mounted into it to persist
	// pulled container images across test runs.
	KINDNodeCache bool `json:"kindNodeCache"`
	// Whether or not to start the KUDO controller for the tests.
	StartKUDO         bool
	// If set, do not delete the resources after running the tests (implies SkipClusterDelete).
	SkipDelete bool `json:"skipDelete"`
	// If set, do not delete the mocked control plane or kind cluster.
	SkipClusterDelete bool `json:"skipClusterDelete"`
	// Override the default assertion timeout of 30 seconds (in seconds).
	Timeout int
}

type Command struct {
	// The command and argument to run as a string.
	Command string `json:"command"`
	// If set, the `--namespace` flag will be appended to the command with the namespace to use.
	Namespaced bool `json:"namespaced"`
	// If set, failures will be ignored.
	IgnoreFailure bool `json:"ignoreFailure"`
}
```

A configuration file can be provided to `kubectl kudo test` using the `--config` argument.

### Test suite directory structure

Test suites are defined in an easy to understand directory structure:

* `tests/`: a top-level directory for all of the test cases, this will typically sit in the same directory as the Operator.
* `tests/$test_case_name/`: each individual test case gets its own directory that contains the test steps. Files in the test case directory are evaluated in alphabetical order: the first file is run, the test harness waits for the expected result to be true (or fail) and then does the same with the next test step.
* `tests/$test_case_name/$index-$test_step.yaml`: a test step called `$test_step` with index `$index`, since it is the first index it gets applied first.

```
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
  - image: alpine
    command:
    - /bin/tail
    - -f
    - /dev/null
```

* `tests/$test_case_name/$index-assert.yaml`: the file called `$index-assert` defines the state that the harness should wait for before continuing. If the state is not true after the timeout, then the entire test case is considered failed.

```
apiVersion: v1
kind: Pod
metadata:
  name: test
status:
  phase: Running
```

* `tests/$test_case_name/$index-errors.yaml`: the file called `$index-errors` defines invalid states that immediately mark a test step failed (and therefore the entire test case).

```
apiVersion: v1
kind: Pod
metadata:
  name: test
status:
  phase: Error
---
apiVersion: v1
kind: Pod
metadata:
  name: test
status:
  phase: Completed
```


Test steps can also be defined in directories instead of a single file - the test harness will look for both YAML files and directories.

### Test step file structure

Each test step has a test step file and an assertion file. By default (even if no assertion file is provided), all resources defined in the test step file are expected to come to a ready state, unless there is a matching object in the assertion file that defines a different expected state.

#### TestStep object

When searching a test step file, if a `TestStep` object is found, it includes settings to apply to the step. This object is not required - if it is not specified then defaults are used. No more than one `TestStep` object should be defined in a given test step.

```
type TestStep struct {
    // The type meta object, should always be a GVK of kudo.dev/v1beta1/TestStep.
    TypeMeta
    // Override the default metadata. Set labels or override the test step name.
    ObjectMeta

    // Objects to delete at the beginning of the test step.
    Delete []ObjectReference

    // Kubectl specifies a list of kubectl commands to run at the beginning of the test step.
    Kubectl []string `json:"kubectl"`

    // Commands to run prior at the beginning of the test step.
    Commands []Command `json:"commands"`

    // Indicates that this is a unit test - safe to run without a real Kubernetes cluster.
    UnitTest bool

    // Allowed environment labels
    // Disallowed environment labels
}

// ObjectReference is a Kubernetes object reference with added labels to allow referencing
// objects by label.
type ObjectReference struct {
	corev1.ObjectReference `json:",inline"`
	// Labels to match on.
	Labels map[string]string
}
```

Using a `TestStep`, it is possible to skip certain test steps if conditions are not met, e.g., only run a test step on GKE or on clusters with more than three nodes.

The `Delete` list can be used to specify objects to delete prior to running the tests. If `Labels` are set in an ObjectReference, all resources matching the labels and specified kind will be deleted.

A `TestStep` is also able to invoke kubectl commands or plugins by specifying a list of commands in the `kubectl` setting, e.g.:

```
apiVersion: kudo.dev/v1beta1
kind: TestStep
kubectl:
- apply -f ./testdata/pod.yaml
```

Any resources created or updated in a kubectl step can be asserted on just like any other resource created in a test step. The commands will be executed in order and the test step will be considered failed if kubectl does not exit with status `0`.

#### Assertion files

The assertion file contains one or more Kubernetes resources to watch the Kubernetes API for. For each object, it checks the API for an object with the same kind, name, and metadata and waits for it to have a state matching what is defined in the assertion files.

For example, given an assertion file containing:

```
apiVersion: v1
kind: Pod
metadata:
  name: test
status:
  phase: Completed
```

The test harness will wait for the `Pod` with name `test` in the namespace generated for the test case to have `status.phase` equal to `Completed` - or return an error if the timeout expires before the resource has the correct state.

##### Resources that have non-deterministic names

Because some resource types have non-deterministic names (for example, the `Pods` created for a `Deployment`), if a resource has no name then the harness will list the resources of that type and wait for a resource of that type to match the state defined.

For example, given an assertion file containing:

```
apiVersion: v1
kind: Pod
metadata:
  labels:
    app: kafka
status:
  phase: Completed
```

The test harness will wait for all `Pods` with label `app=kafka` in the namespace generated for the test case to have `status.phase` equal to `Completed` - or return an error if the timeout expires before the resource has the correct state.

##### TestAssert object

When searching the assertion file for a test step, if a `TestAssert` object is found, it includes settings to apply to the assertions. This object is not required - if it is not specified then defaults are used. No more than one `TestAssert` should be defined in a test assertion.

```
type TestAssert struct {
    // The type meta object, should always be a GVK of kudo.dev/v1beta1/TestAssert.
    TypeMeta
    // Override the default timeout of 30 seconds (in seconds).
    Timeout int
}
```

#### Commands

If commands are set in the test suite or step they will be run at the beginning of the test suite or step, respectively. Commands are run in order and are typed so that functionality can be extended.

The command is split on spaces (while respecting quoted strings, e.g., using [shlex](https://godoc.org/github.com/google/shlex)). If the command has the `Namespaced` setting set, then the `--namespace` flag will be added with the test case namespace (or "default" for the the test suite).

### Results collection

Successful test cases will produce a log of all objects created and their state when the test step was finalized. Failed test cases will highlight the difference in expected state in an easy to read manner. Logs will also be collected from pods in failed test cases.

### Garbage collection

In order for a test case to be considered successful, all resources created by it must delete cleanly. The test harness will track all resources that are created by the test case and then delete any resources that still exist at the end of the test case. If they do not delete in a timely fashion, then the test case is failed - this has the effect of preventing resource leaks from running the test suite and also ensuring that Operators can be uninstalled cleanly.

## Graduation Criteria

* Integration into KUDO Operators's pull request pipeline.
* Integration into KUDO's pull request pipeline.
* CLI for running tests.
* Plugin for [Sonobuoy](https://github.com/heptio/sonobuoy/blob/master/docs/plugins.md) available.
* Adoption by two stable Operators.

## Implementation History

- 2019/05/03 - Initial draft.

## Alternatives

Controllers are typically tested using test libraries in the same language that they are written in. While these libraries and examples can provide good inspiration and insights into how to test Operators, they depart from the declarative spirit of KUDO making them unsuitable for use as the user-facing interface for writing tests.

* [Kubernetes e2e test operator](https://godoc.org/k8s.io/kubernetes/test/e2e/operator) provides a methods that interact with Kubernetes resources and wait for certain Kubernetes state. It also supports conditionally running tests and collecting logs and results from pods and nodes.
* Unit tests can be written using the [Kubernetes fake clientset](https://godoc.org/k8s.io/client-go/kubernetes/fake) without needing a Kubernetes API at all - allowing easy testing of expected state transitions in a controller.
* The [controller-runtime](https://godoc.org/sigs.k8s.io/controller-runtime/pkg) provides test machinery that makes it easy to integration test controllers without a running Kubernetes cluster. The KUDO project itself uses these extensively.
* The [Kubernetes command-line integration test suite](https://github.com/kubernetes/kubernetes/tree/master/test/cmd) is a BASH-driven integration test suite that uses kubectl commands to run tests. This could be a suitable option as it is not a specialized programming language, but it is an imperative method of testing which may not be the right UX for KUDO.
* [Terratest](https://godoc.org/github.com/gruntwork-io/terratest/modules/k8s) is a Go-based testing harness that provides methods for interacting with Kubernetes resources and waiting for them to be ready.
* [metacontroller](https://github.com/GoogleCloudPlatform/metacontroller/blob/master/examples/daemonjob/test.sh) does not provide a test harness out of the box, but many of their examples use a bash script with kubectl commands to compose simple tests.
* [Terraform provider acceptance tests](https://www.terraform.io/docs/extend/testing/acceptance-tests/testcase.html) are authored by providing a series of configurations to apply with expected states and provided some inspiration for this design.

[Helm charts](https://github.com/helm/helm/blob/master/docs/chart_tests.md) use a scheme for running tests where tests are defined as Kubernetes Jobs that are run and the result is determined by the Job status. This methodology is compatible with KUDO and can be seen applied in the Zookeeper Operator's [validation job](https://github.com/kudobuilder/operators/blob/2b1151eca761c0fbe61474ba44b0bdaa4f80a0fb/repo/stable/zookeeper/versions/0/zookeeper-operatorversion.yaml#L161-L188). This document does not supercede this technique (tests written for KUDO Operators can easily incorporate these validation stages), the machinery provided here will be able to easily incorporate these tests to improve test quality.

[OPA's testing harness](https://www.openpolicyagent.org/docs/v0.10.7/how-do-i-test-policies/) takes a similar approach to testing JSON objects, by allowing evaluating OPA policies and then asserting on certain attributes or responses given mock inputs and OPA in general is a good example of testing declarative resources (see [terraform testing](https://www.openpolicyagent.org/docs/latest/terraform/)). If we find the proposed scheme for asserting on resource attributes is not powerful enough, then OPA policies may be a good approach for authoring assertions.

## Infrastructure Needed

A CI system and cloud infrastructure for running the test suite are required (see [0004-add-testing-infrastructure](https://github.com/kudobuilder/kudo/blob/master/keps/0004-add-testing-infrastructure.md)).
