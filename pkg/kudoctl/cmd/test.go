package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	harness "github.com/kudobuilder/kuttl/pkg/apis/testharness/v1beta1"
	"github.com/kudobuilder/kuttl/pkg/test"
	testutils "github.com/kudobuilder/kuttl/pkg/test/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	testExample = `  Run tests configured by kudo-test.yaml:
    kubectl kudo test

  Load a specific test configuration:
    kubectl kudo test --config test.yaml

  Run tests against an existing Kubernetes cluster:
    kubectl kudo test ./test/integration/

  Run tests against an existing Kubernetes cluster, and install KUDO, manifests, and CRDs for the tests:
    kubectl kudo test --crd-dir ./config/crds/ --manifests-dir ./test/manifests/ ./test/integration/

  Run a Kubernetes control plane and KUDO and install manifests and CRDs for the running tests:
    kubectl kudo test --start-control-plane --start-kudo --crd-dir ./config/crds/ --manifests-dir ./test/manifests/ ./test/integration/
`
)

// newTestCmd creates the test command for the CLI
func newTestCmd() *cobra.Command {
	configPath := ""
	crdDir := ""
	manifestDirs := []string{}
	testToRun := ""
	startControlPlane := false
	startKIND := false
	kindConfig := ""
	kindContext := ""
	startKUDO := false
	skipDelete := false
	skipClusterDelete := false
	parallel := 0
	artifactsDir := ""
	reportFormat := ""

	options := harness.TestSuite{}

	testCmd := &cobra.Command{
		Use:   "test [flags]... [test directories]...",
		Short: "Test KUDO and Operators.",
		Long: `Runs integration tests against a Kubernetes cluster.

The test operator supports connecting to an existing Kubernetes cluster or it can start a Kubernetes API server during the test run.
It can also start up KUDO and apply manifests before running the tests. If no arguments are provided, the test harness will attempt to 
load the test configuration from kudo-test.yaml.

For more detailed documentation, visit: https://kudo.dev/docs/testing`,
		Example: testExample,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			flags := cmd.Flags()

			options.TestDirs = args

			// If a config is not set and kudo-test.yaml exists, set configPath to kudo-test.yaml.
			if configPath == "" {
				if _, err := os.Stat("kudo-test.yaml"); err == nil {
					configPath = "kudo-test.yaml"
				} else {
					return errors.New("kudo-test.yaml not found, provide either --config or arguments indicating the tests to load")
				}
			}

			// Load the configuration YAML into options.
			if configPath != "" {
				objects, err := testutils.LoadYAMLFromFile(configPath)
				if err != nil {
					return err
				}

				for _, obj := range objects {
					kind := obj.GetObjectKind().GroupVersionKind().Kind

					if kind == "TestSuite" {
						options = *obj.(*harness.TestSuite)
					} else {
						log.Println(fmt.Errorf("unknown object type: %s", kind))
					}
				}
			}

			// Override configuration file options with any command line flags if they are set.

			if isSet(flags, "crd-dir") {
				options.CRDDir = crdDir
			}

			if isSet(flags, "manifest-dir") {
				options.ManifestDirs = manifestDirs
			}

			if isSet(flags, "start-control-plane") {
				options.StartControlPlane = startControlPlane
			}

			if isSet(flags, "start-kind") {
				options.StartKIND = startKIND
			}

			if isSet(flags, "kind-config") {
				options.StartKIND = true
				options.KINDConfig = kindConfig
			}

			if isSet(flags, "kind-context") {
				options.KINDContext = kindContext
			}

			if options.KINDContext == "" {
				options.KINDContext = harness.DefaultKINDContext
			}

			if options.StartControlPlane && options.StartKIND {
				return errors.New("only one of --start-control-plane and --start-kind can be set")
			}

			// if isSet(flags, "start-kudo") {
			// TODO (kensipe): switch to a new way to start kudo (outside of kuttl)
			// }

			if isSet(flags, "skip-delete") {
				options.SkipDelete = skipDelete
			}

			if isSet(flags, "skip-cluster-delete") {
				options.SkipClusterDelete = skipClusterDelete
			}

			if isSet(flags, "parallel") {
				options.Parallel = parallel
			}

			if isSet(flags, "artifacts-dir") {
				options.ArtifactsDir = artifactsDir
			}

			if isSet(flags, "report") {
				options.ReportFormat = strings.ToLower(reportFormat)
			}

			if len(args) != 0 {
				options.TestDirs = args
			}

			if len(options.TestDirs) == 0 {
				return errors.New("no test directories provided, please provide either --config or test directories on the command line")
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			testutils.RunTests("kudo", testToRun, options.Parallel, func(t *testing.T) {
				harness := test.Harness{
					TestSuite: options,
					T:         t,
				}

				s, _ := json.MarshalIndent(options, "", "  ")
				fmt.Printf("Running integration tests with following options:\n%s\n", string(s))

				harness.Run()
			})
		},
	}

	testCmd.Flags().StringVar(&configPath, "config", "", "Path to file to load test settings from (must not be set with any other arguments).")
	testCmd.Flags().StringVar(&crdDir, "crd-dir", "", "Directory to load CustomResourceDefinitions from prior to running the tests.")
	testCmd.Flags().StringSliceVar(&manifestDirs, "manifest-dir", []string{}, "One or more directories containing manifests to apply before running the tests.")
	testCmd.Flags().StringVar(&testToRun, "test", "", "If set, the specific test case to run.")
	testCmd.Flags().BoolVar(&startControlPlane, "start-control-plane", false, "Start a local Kubernetes control plane for the tests (requires etcd and kube-apiserver binaries, cannot be used with --start-kind).")
	testCmd.Flags().BoolVar(&startKIND, "start-kind", false, "Start a KIND cluster for the tests (cannot be used with --start-control-plane).")
	testCmd.Flags().StringVar(&kindConfig, "kind-config", "", "Specify the KIND configuration file path (implies --start-kind, cannot be used with --start-control-plane).")
	testCmd.Flags().StringVar(&kindContext, "kind-context", "", "Specify the KIND context name to use (default: kind).")
	testCmd.Flags().StringVar(&artifactsDir, "artifacts-dir", "", "Directory to output kind logs to (if not specified, the current working directory).")
	testCmd.Flags().StringVar(&reportFormat, "report", "", "Specify JSON|XML for report.  Report location determined by --artifacts-dir.")
	testCmd.Flags().BoolVar(&startKUDO, "start-kudo", false, "Start KUDO during the test run.")
	testCmd.Flags().BoolVar(&skipDelete, "skip-delete", false, "If set, do not delete resources created during tests (helpful for debugging test failures, implies --skip-cluster-delete).")
	testCmd.Flags().BoolVar(&skipClusterDelete, "skip-cluster-delete", false, "If set, do not delete the mocked control plane or kind cluster.")
	// The default value here is only used for the help message. The default is actually enforced in RunTests.
	testCmd.Flags().IntVar(&parallel, "parallel", 8, "The maximum number of tests to run at once.")

	return testCmd
}

// isSet returns true if a flag is set on the command line.
func isSet(flagSet *pflag.FlagSet, name string) bool {
	found := false

	flagSet.Visit(func(flag *pflag.Flag) {
		if flag.Name == name {
			found = true
		}
	})

	return found
}
