package install

import (
	"testing"
)

func TestInstallFrameworks(t *testing.T) {

	// For test case #1
	expectedNoArgumentErrors := []string{
		"no argument provided",
	}

	// For test case #2
	options := DefaultOptions
	options.PackageVersion = "0.0"
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
		err := installFrameworks(tt.args, options)
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
