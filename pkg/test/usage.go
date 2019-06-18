package test

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

var manifestsDir *string
var crdDir *string
var startKUDO *bool
var startControlPlane *bool

// Usage prints the KUDO test framework help text.
func Usage() {
	fmt.Printf("Usage: %s [OPTION]... [TEST DIRECTORY]...\n\n", os.Args[0])

	fmt.Printf("Runs integration tests against a Kubernetes cluster.\n\n")

	fmt.Printf("The test framework supports connecting to an existing Kubernetes cluster or it can start a Kubernetes API server during the test run.\n\n")

	fmt.Printf("It can also start up KUDO and apply manifests before running the tests.\n\n")

	fmt.Printf("KUDO test framework options:\n\n")

	goTestFlags := []*flag.Flag{}

	flag.VisitAll(func(f *flag.Flag) {
		if strings.HasPrefix(f.Name, "ginkgo.") {
			return
		}

		if strings.HasPrefix(f.Name, "test.") {
			goTestFlags = append(goTestFlags, f)
			return
		}

		flagValue := fmt.Sprintf("-%s", f.Name)
		if f.DefValue != "" {
			flagValue = fmt.Sprintf("%s=%s", flagValue, f.DefValue)
		}

		fmt.Printf("      %s: %s\n", flagValue, f.Usage)
	})

	fmt.Printf("\nExample usage:\n\n")
	fmt.Println("   Run tests against an existing Kubernetes cluster:")
	fmt.Printf("      %s ./pkg/test/test_data/\n\n", os.Args[0])
	fmt.Println("   Run tests against an existing Kubernetes cluster, and install KUDO, manifests, and CRDs for the tests:")
	fmt.Printf("      %s -kudo -crds ./config/crds/ -manifests ./config/samples/test-framework/ ./pkg/test/test_data/\n\n", os.Args[0])
	fmt.Println("   Run a Kubernetes control plane and KUDO and install manifests and CRDs for the running tests:")
	fmt.Printf("      %s -control-plane -kudo -crds ./config/crds/ -manifests ./config/samples/test-framework/ ./pkg/test/test_data/\n\n", os.Args[0])
	fmt.Printf("Note that in order to use the Go test cache, the test harness should be invoked using go test and configured with environment variables:\n\n")
	fmt.Printf("      START_CONTROL_PLANE=true START_KUDO=true MANIFESTS_DIR=../../config/samples/test-framework/ CRD_DIR=../../config/crds/ TESTS_DIR=../../pkg/test/test_data go test ./cmd/test/... -tags build_harness\n\n")
	fmt.Println("For more detailed documentation, visit: https://kudo.dev/docs/testing")

	fmt.Printf("\nGo test options:\n\n")

	for _, f := range goTestFlags {
		fmt.Printf("      -%s: %s\n", f.Name, f.Usage)
	}
}

// RegisterFlags registers all of the flags needed for the KUDO test framework. It should be called in an init method.
func RegisterFlags() {
	flag.Usage = Usage
	manifestsDir = flag.String("manifests", "", "A directory containing manifests to apply before running the tests (env: $FRAMEWORKS_DIR).")
	crdDir = flag.String("crds", "", "Directory containing CRDs to install into the cluster (env: $CRD_DIR).")
	startKUDO = flag.Bool("kudo", false, "Start KUDO during the test run (env: $START_KUDO).")
	startControlPlane = flag.Bool("control-plane", false, "Start a local Kubernetes control plane for the tests (requires etcd and kube-apiserver binaries, implies -kudo, env: $START_CONTROL_PLANE).")
}

// HarnessFromFlags returns a test Harness instantiated from command line flags or environment variables.
func HarnessFromFlags(t *testing.T) *Harness {
	if *manifestsDir == "" {
		*manifestsDir = os.Getenv("MANIFESTS_DIR")
	}

	if *crdDir == "" {
		*crdDir = os.Getenv("CRD_DIR")
	}

	if !*startKUDO {
		*startKUDO, _ = strconv.ParseBool(os.Getenv("START_KUDO"))
	}

	if !*startControlPlane {
		*startControlPlane, _ = strconv.ParseBool(os.Getenv("START_CONTROL_PLANE"))
	}

	testDirs := flag.Args()
	if len(testDirs) == 0 {
		testDirs = strings.Split(os.Getenv("TESTS_DIR"), ",")
	}

	return &Harness{
		T:                 t,
		TestDirs:          testDirs,
		ManifestsDir:      *manifestsDir,
		CRDDir:            *crdDir,
		StartKUDO:         *startKUDO,
		StartControlPlane: *startControlPlane,
	}
}
