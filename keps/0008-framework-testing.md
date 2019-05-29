---
kep-number: 8
title: Framework Testing
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

# Framework Testing

## Table of Contents

  * [Summary](#summary)
  * [Motivation](#motivation)
     * [Goals](#goals)
     * [Non-Goals](#non-goals)
  * [User Stories](#user-stories)
     * [KUDO Developers](#kudo-developers)
     * [Framework Developer](#framework-developer)
     * [Cluster Administrator](#cluster-administrator)
     * [Application Operator](#application-operator)
  * [Risks and Mitigations](#risks-and-mitigations)
  * [Proposal](#proposal)
     * [Definitions](#definitions)
     * [Running the tests](#running-the-tests)
     * [Directory structure](#directory-structure)
     * [Test case file structure](#test-case-file-structure)
        * [TestCase object](#testcase-object)
        * [TestAssert object](#testassert-object)
        * [Test constraints](#test-constraints)
     * [Results collection](#results-collection)
     * [Garbage collection](#garbage-collection)
  * [Graduation Criteria](#graduation-criteria)
  * [Implementation History](#implementation-history)
  * [Alternatives](#alternatives)
  * [Infrastructure Needed](#infrastructure-needed)

## Summary

In order to ensure the reliability of Frameworks built on KUDO, it is important that developers of Frameworks are able to simply author and run tests that validate that their Frameworks are running correctly. Tests can be run in CI to verify changes, by Framework developers in their development cycle, and by cluster administrators to verify that Frameworks are fully functional in their Kubernetes clusters, across a variety of configurations.

This document outlines a simple, declarative test harness for authoring and running acceptance tests for KUDO Frameworks.

## Motivation

### Goals

* Allow Framework developers to write acceptance and unit tests for Frameworks.
* Allow Application Operators and Cluster Administrators to write tests that validate their customizations to Frameworks.
* Run tests for Frameworks on pull requests for Frameworks.
* Support running tests via a `kubectl kudo` subcommand against any cluster.
* Integration with Sonobuoy to validate Frameworks in any cluster.
* Ensure that KUDO Framework developers have a consistent development experience - from writing their operators to testing them.

### Non-Goals

* Outlining testing policies for Frameworks (e.g., minimum coverage to graduate to "stable").
* Provisioning Kubernetes test infrastructure.

## User Stories

#### KUDO Developers

As the KUDO development team, I...

* *Must* be able to validate that KUDO releases do not break existing Frameworks.
* *Must* be able to validate that Framework contributions do not break existing Frameworks.
* *Must* be able to validate Frameworks in many different Kubernetes environments to ensure compatibility.

#### Framework Developer

As a Framework Developer, I...

* *Must* be able to validate that my changes do not break my Frameworks and that my features work correctly.
* *Must* be able to write tests for many different valid (and invalid) configurations.
* *Want to* be able to author tests in a declarative language, without resorting to writing code in a normal programming language.
* *Must* be able to easily understand test results and failures.
* *Must* be able to incorporate KUDO Framework plans into my tests.

#### Cluster Administrator

As a Cluster Administrator, I...

* *Must* be able to easily validate that Frameworks work well in my cluster.
* *Want to* be able to author my own tests for known failure modes or critical features.

#### Application Operator

As an Application Operator, I...

* *Must* be able to validate that my Instances are working correctly.

## Risks and Mitigations

* Perhaps the most important consideration when building any testing pipeline is ensuring that developer velocity is not impacted. This is especially important for KUDO, since the developer experience for Framework developers is critical to the success of the project. To mitigate this, we need to optimize the runtime of the tests and pull request flow. Framework contributions should only run tests for the Frameworks that are affected by the pull request. Tests should have a fixed upper time limit to prevent any new tests from causing excessive delays. Tests fail early.
* If the test harness is not designed well, contributions to the project become very difficult or the harness does not get adopted by developers. We provide a minimal, declarative format for authoring tests to encourage adoption and provide simple ways to execute the tests to ensure that it is easy to get started.
* The test harness is not valuable if no tests are written. This means Framework developers need to write tests! We encourage test authoring by writing good documentation, integrating the test harness into the CI/CD pipeline, and potentially enacting policies around testing for official Frameworks. We can also advocate for testing via blogs and interactions with Framework developers.
* As we take a declarative approach to testing, it's important to ensure that the abstractions built are not leaky. Leaky abstractions lead to greater complexity and make the test suite hard to use. We decrease this risk by narrowing the scope to focus on the state of Kubernetes objects in the Kubernetes API, refering to prior work (see [Alternatives](#Alternatives)), and building a minimal solution. We also will write tests for existing Frameworks and work with authors of new Frameworks to ensure that the solution is generally useful.

## Proposal

Tests will be authored by defining Kubernetes objects to apply and Kubernetes state to wait for. Each test will run in its own namespace (tests can run concurrently), within each test, test cases run sequentially. This allows test authors to model state transitions of Frameworks as different configurations are applied.

### Definitions

* `Test case`: a set of manifests to apply and the expected state. Test cases run sequentially within a Test.
* `Assertion`: the expected state of a test case.
* `Test`: a complete acceptance test, consisting of multiple test cases. Tests run concurrently to each other.
* `Framework`: A KUDO Framework.
* `Harness`: the tool that runs the tests.

### Running the tests

Tests will be invoked via a CLI tool that runs the tests against the current Kubernetes context. It will be packaged with the default set of tests from the KUDO Frameworks repository using go-bindata, with the ability to provide alternative directories containing tests to use.

The tool will enumerate each test (group of test cases) and run them concurrently in batches. Each test will run in its own namespace (care must be taken that cluster-level resources do not collide with other tests [??? TODO: solvable?]) which will be deleted after the test has been completed.

Each test consists of a directory containing test cases and their expected results ("assertions"). The test cases within a test are run sequentially, waiting for the assertions to be true, a timeout to pass, or a failure condition to be met.

Once all of the test cases in a test have run, a report for the test is generated and all of its resources are deleted.

The test harness can also be run in a unit testing mode, where it spins up a dummy Kubernetes API server and runs any tests marked as unit tests.

### Directory structure

Tests are defined in an easy to understand directory structure:

* `tests/`: a top-level test directory for all of the tests, this will typically sit in the same directory as the Framework.
* `tests/$test_name/`: each individual test gets its own directory that contains the objects to create and the expected objects. Files in the test directory are evaluated in alphabetical order: the first file is run, the test harness waits for the expected result to be true (or fail) and then does the same with the next test case.
* `tests/$test_name/$index-$test_case.yaml`: a test case called `$test_case` with index `$index`, since it is the first index it gets applied first.

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

* `tests/$test_name/$index-assert.yaml`: the file called `$index-assert` defines the state that the harness should wait for before continuing. If the state is not true after the timeout, then the test is considered failed.

```
apiVersion: v1
kind: Pod
metadata:
  name: test
status:
  phase: Running
```

* `tests/$test_name/$index-errors.yaml`: the file called `$index-errors` defines invalid states that immediately mark a test failed.

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


Test cases can also be defined in directories instead of a single file - the test harness will look for both YAML files and directories.

### Test case file structure

Each test case has a test case file and an assertion file. By default (even if no assertion file is provided), all resources defined in the test case file are expected to come to a ready state, unless there is a matching object in the assertion file that defines a different expected state.

#### TestCase object

When searching a test case file, if a `TestCase` object is found, it includes settings to apply to the case. This object is not required - if it is not specified then defaults are used. No more than one `TestCase` should be defined in a test case.

```
type TestCase struct {
    // The type meta object, should always be a GVK of kudo.k8s.io/v1alpha1/TestCase.
    TypeMeta
    // Override the default metadata. Set labels or override the test case name.
    ObjectMeta
    // Objects to delete at the beginning of the test case.
    Delete []corev1.ObjectReference
}
```

#### TestAssert object

When searching the assertion file for a test case, if a `TestAssert` object is found, it includes settings to apply to the assertions. This object is not required - if it is not specified then defaults are used. No more than one `TestAssert` should be defined in a test assertion.

```
type TestAssert struct {
    // The type meta object, should always be a GVK of kudo.k8s.io/v1alpha1/TestAssert.
    TypeMeta
    // Override the default timeout of 300 seconds (in seconds).
    Timeout int
}
```

#### Test constraints

It is possible to skip certain tests if conditions are not met, e.g., only run a test on GKE or on clusters with more than three nodes.

```
type TestConstraint struct {
    // The type meta object, should always be a GVK of kudo.k8s.io/v1alpha1/TestConstraint.
    TypeMeta
    // Indicates that this is a unit test - safe to run without a real Kubernetes cluster.
    UnitTest bool
    // Allowed environment labels
    // Disallowed environment labels
}
```

### Results collection

Successful tests will produce a log of all objects created and their state when the test case was finalized. Failed tests will highlight the difference in expected state in an easy to read manner. Logs will also be collected from pods in failed tests.

### Garbage collection

In order for a test to be considered successful, all resources created by it must delete cleanly. The test harness will track all resources that are created by the test and then delete any resources that still exist at the end of the test. If they do not delete in a timely fashion, then the test is failed - this has the effect of preventing resource leaks from running the tests and also ensuring that Frameworks can be uninstalled cleanly.

## Graduation Criteria

* Integration into KUDO Frameworks's pull request pipeline.
* Integration into KUDO's pull request pipeline.
* CLI for running tests.
* Plugin for [Sonobuoy](https://github.com/heptio/sonobuoy/blob/master/docs/plugins.md) available.
* Adoption by two stable Frameworks.

## Implementation History

- 2019/05/03 - Initial draft.

## Alternatives

Controllers are typically tested using test libraries in the same language that they are written in. While these libraries and examples can provide good inspiration and insights into how to test Frameworks, they depart from the declarative spirit of KUDO making them unsuitable for use as the user-facing interface for writing tests.

* [Kubernetes e2e test framework](https://godoc.org/k8s.io/kubernetes/test/e2e/framework) provides a methods that interact with Kubernetes resources and wait for certain Kubernetes state. It also supports conditionally running tests and collecting logs and results from pods and nodes.
* Unit tests can be written using the [Kubernetes fake clientset](https://godoc.org/k8s.io/client-go/kubernetes/fake) without needing a Kubernetes API at all - allowing easy testing of expected state transitions in a controller.
* The [controller-runtime](https://godoc.org/sigs.k8s.io/controller-runtime/pkg) provides test machinery that makes it easy to integration test controllers without a running Kubernetes cluster. The KUDO project itself uses these extensively.
* The [Kubernetes command-line integration test suite](https://github.com/kubernetes/kubernetes/tree/master/test/cmd) is a BASH-driven integration test suite that uses kubectl commands to run tests. This could be a suitable option as it is not a specialized programming language, but it is an imperative method of testing which may not be the right UX for KUDO.
* [Terratest](https://godoc.org/github.com/gruntwork-io/terratest/modules/k8s) is a Go-based testing harness that provides methods for interacting with Kubernetes resources and waiting for them to be ready.
* [metacontroller](https://github.com/GoogleCloudPlatform/metacontroller/blob/master/examples/daemonjob/test.sh) does not provide a test harness out of the box, but many of their examples use a bash script with kubectl commands to compose simple tests.
* [Terraform provider acceptance tests](https://www.terraform.io/docs/extend/testing/acceptance-tests/testcase.html) are authored by providing a series of configurations to apply with expected states and provided some inspiration for this design.

[Helm charts](https://github.com/helm/helm/blob/master/docs/chart_tests.md) use a scheme for running tests where tests are defined as Kubernetes Jobs that are run and the result is determined by the Job status. This methodology is compatible with KUDO and can be seen applied in the Zookeeper Framework's [validation job](https://github.com/kudobuilder/frameworks/blob/2b1151eca761c0fbe61474ba44b0bdaa4f80a0fb/repo/stable/zookeeper/versions/0/zookeeper-frameworkversion.yaml#L161-L188). This document does not supercede this technique (tests written for KUDO Frameworks can easily incorporate these validation stages), the machinery provided here will be able to easily incorporate these tests to improve test quality.

[OPA's testing harness](https://www.openpolicyagent.org/docs/v0.10.7/how-do-i-test-policies/) takes a similar approach to testing JSON objects, by allowing evaluating OPA policies and then asserting on certain attributes or responses given mock inputs and OPA in general is a good example of testing declarative resources (see [terraform testing](https://www.openpolicyagent.org/docs/latest/terraform/)). If we find the proposed scheme for asserting on resource attributes is not powerful enough, then OPA policies may be a good approach for authoring assertions.

## Infrastructure Needed

A CI system and cloud infrastructure for running the tests are required (see [0004-add-testing-infrastructure](https://github.com/kudobuilder/kudo/blob/master/keps/0004-add-testing-infrastructure.md)).
