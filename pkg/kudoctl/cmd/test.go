package cmd

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"testing"

	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/test"
	testutils "github.com/kudobuilder/kudo/pkg/test/utils"

	"github.com/spf13/cobra"
)

var (
	testExample = `
      Run tests configured by kudo-test.yaml:
            kubectl kudo test

      Load a specific test configuration:
            kubectl kudo test --config test.yaml

      Run tests against an existing Kubernetes cluster:
            kubectl kudo test ./pkg/test/test_data/

      Run tests against an existing Kubernetes cluster, and install KUDO, manifests, and CRDs for the tests:
            kubectl kudo test --crd-dir ./config/crds/ --manifests-dir ./config/samples/test-framework/ ./pkg/test/test_data/

      Run a Kubernetes control plane and KUDO and install manifests and CRDs for the running tests:
            kubectl kudo test --start-control-plane --start-kudo --crd-dir ./config/crds/ --manifests-dir ./config/samples/test-framework/ ./pkg/test/test_data/
`
)

// newTestCmd creates the test command for the CLI
func newTestCmd() *cobra.Command {
	configPath := ""
	options := kudo.TestSuite{}
	defaults := kudo.TestSuite{
		TestDirs: []string{},
	}

	testCmd := &cobra.Command{
		Use:   "test [flags]... [test directories]...",
		Short: "-> Test KUDO and Frameworks.",
		Long: `Runs integration tests against a Kubernetes cluster.

The test framework supports connecting to an existing Kubernetes cluster or it can start a Kubernetes API server during the test run.

It can also start up KUDO and apply manifests before running the tests.

If no arguments are provided, the test harness will attempt to load the test configuration from kudo-test.yaml.

For more detailed documentation, visit: https://kudo.dev/docs/testing`,
		Example: testExample,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			options.TestDirs = args

			if configPath == "" && reflect.DeepEqual(options, defaults) {
				if _, err := os.Stat("kudo-test.yaml"); err == nil {
					configPath = "kudo-test.yaml"
				} else {
					return fmt.Errorf("kudo-test.yaml not found, provide either --config or arguments indicating the tests to load")
				}
			}

			if configPath != "" && !reflect.DeepEqual(options, defaults) {
				return fmt.Errorf("provide either --config or other arguments, but not both")
			}

			if configPath != "" {
				objects, err := testutils.LoadYAML(configPath)
				if err != nil {
					return err
				}

				for _, obj := range objects {
					kind := obj.GetObjectKind().GroupVersionKind().Kind

					if kind == "TestSuite" {
						options = *obj.(*kudo.TestSuite)
					} else {
						log.Println(fmt.Errorf("unknown object type: %s", kind))
					}
				}
			}

			if len(options.TestDirs) == 0 {
				return fmt.Errorf("no test directories provided, please provide either --config or test directories on the command line")
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			testutils.RunTests("kudo", func(t *testing.T) {
				harness := test.Harness{
					TestSuite: options,
					T:         t,
				}

				harness.Run()
			})
		},
	}

	testCmd.Flags().StringVar(&configPath, "config", "", "Path to file to load test settings from (must not be set with any other arguments).")
	testCmd.Flags().StringVar(&options.CRDDir, "crd-dir", "", "Directory to load CustomResourceDefinitions from prior to running the tests.")
	testCmd.Flags().StringVar(&options.ManifestsDir, "manifests-dir", "", "A directory containing manifests to apply before running the tests.")
	testCmd.Flags().BoolVar(&options.StartControlPlane, "start-control-plane", false, "Start a local Kubernetes control plane for the tests (requires etcd and kube-apiserver binaries, implies --start-kudo).")
	testCmd.Flags().BoolVar(&options.StartKUDO, "start-kudo", false, "Start KUDO during the test run.")

	return testCmd
}
