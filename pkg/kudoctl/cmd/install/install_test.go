package install

import (
	"io/ioutil"
	"testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
	"github.com/spf13/cobra"
)

func TestInstallCmd(t *testing.T) {

	// Default for test case #1
	cmdDefault := &cobra.Command{}

	expectedDefaultErrors := []string{
		"get flag: flag accessed but not defined: kubeconfig",
	}

	// For test case #2
	cmdNoKubeconfigFlagDefined := &cobra.Command{}
	expectedNoKubeconfigFlagDefinedErrors := []string{
		"get flag: flag accessed but not defined: kubeconfig",
	}

	// For test case #3
	cmdWrongDirKubeConfigFlag := &cobra.Command{}
	cmdWrongDirKubeConfigFlag.Flags().StringVar(&vars.KubeConfigPath, "kubeconfig", "", "Usage")
	vars.KubeConfigPath = "/tmp"
	index := []byte(`apiVersion: v1
entries:
  kafka:
  - apiVersion: v1
    appVersion: 2.4.0
    name: kafka
    urls:
    - https://kudo-test-repo.storage.googleapis.com/kafka-0.1.0.tgz
    version: 0.1.0`)
	_ = ioutil.WriteFile("/tmp/index.yaml", index, 0644)
	installCmdArgs := []string{"test", "--kubeconfig=" + vars.KubeConfigPath}
	expectedEmptyKubeConfigFlagErrors := []string{
		"could not install framework(s): getting config failed: Error loading config file \"/tmp\": read /tmp: is a directory",
	}

	tests := []struct {
		cmd  *cobra.Command
		args []string
		err  []string
	}{
		{cmdDefault, nil, expectedDefaultErrors},                                       // 1
		{cmdNoKubeconfigFlagDefined, nil, expectedNoKubeconfigFlagDefinedErrors},       // 2
		{cmdWrongDirKubeConfigFlag, installCmdArgs, expectedEmptyKubeConfigFlagErrors}, // 3
	}

	for i, tt := range tests {
		i := i
		err := CmdErrorProcessor(tt.cmd, tt.args)
		if err != nil {
			receivedErrorList := []string{err.Error()}
			diff := compareSlice(receivedErrorList, tt.err)
			for _, err := range diff {
				t.Errorf("%d: Found unexpected error: %v", i+1, err)
			}

			missing := compareSlice(tt.err, receivedErrorList)
			for _, err := range missing {
				t.Errorf("%d: Missed expected error: %v", i+1, err)
			}
		}
	}
}

func TestInstallFrameworks(t *testing.T) {

	// For test case #1
	expectedNoArgumentErrors := []string{
		"no argument provided",
	}

	// For test case #2
	vars.KubeConfigPath = ""
	vars.PackageVersion = "0.0"
	installCmdPackageVersionArgs := []string{"one", "two"}
	expectedPackageVersionFlagErrors := []string{
		"--package-version not supported in multi framework install",
	}

	tests := []struct {
		args []string
		err  []string
	}{
		{nil, expectedNoArgumentErrors},                                  // 1
		{installCmdPackageVersionArgs, expectedPackageVersionFlagErrors}, // 2
	}

	for i, tt := range tests {
		i := i
		err := verifyFrameworks(tt.args)
		if err != nil {
			receivedErrorList := []string{err.Error()}
			diff := compareSlice(receivedErrorList, tt.err)
			for _, err := range diff {
				t.Errorf("%d: Found unexpected error: %v", i+1, err)
			}

			missing := compareSlice(tt.err, receivedErrorList)
			for _, err := range missing {
				t.Errorf("%d: Missed expected error: %v", i+1, err)
			}
		}
	}
}

func compareSlice(real, mock []string) []string {
	lm := len(mock)

	var diff []string

	for _, rv := range real {
		i := 0
		j := 0
		for _, mv := range mock {
			i++
			if rv == mv {
				continue
			}
			if rv != mv {
				j++
			}
			if lm <= j {
				diff = append(diff, rv)
			}
		}
	}
	return diff
}
