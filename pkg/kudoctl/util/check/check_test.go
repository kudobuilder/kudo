package check

import (
	"os/user"
	"path/filepath"
	"testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
)

func TestKubeConfigPath(t *testing.T) {
	// first we test that the vars.KubeConfigPath is propagated correctly when resolving the path
	vars.KubeConfigPath = "/tmp/;"
	location, err := getKubeConfigLocation()
	if err != nil {
		t.Errorf("expected kubeconfig path '%v' to be propagated from vars, got error instead %v", vars.KubeConfigPath, err)
	}
	if location != vars.KubeConfigPath {
		t.Errorf("expected kubeconfig path '%v' to be propagated from vars, kubeconfig path instead resolved as %v", vars.KubeConfigPath, location)
	}

	// then we test that default is used when no path is provided in vars
	vars.KubeConfigPath = ""
	usr, _ := user.Current()
	expectedPath := filepath.Join(usr.HomeDir, defaultKubeConfigPath)
	location, err = getKubeConfigLocation()
	if err != nil {
		t.Errorf("expected kubeconfig path '%v', got error instead %v", expectedPath, err)
	}
	if location != expectedPath {
		t.Errorf("expected kubeconfig path '%v', kubeconfig path instead resolved as %v", expectedPath, location)
	}
}

func TestGithubCredentials(t *testing.T) {
	vars.GithubCredentialPath = "/tmp/;"

	testNonExisting := []struct {
		expected string
	}{
		{"failed to find github credential file: stat /tmp/;: no such file or directory"}, // 1
	}

	for _, tt := range testNonExisting {
		actual := GithubCredentials()
		if actual != nil {
			if actual.Error() != tt.expected {
				t.Errorf("non existing test:\nexpected: %v\n     got: %v", tt.expected, actual)
			}
		}
	}

	vars.GithubCredentialPath = ""

	testZero := []struct {
		expected *string
	}{
		{nil}, // 1
	}

	for _, tt := range testZero {
		actual := GithubCredentials()
		if actual != nil {
			t.Errorf("empty path test:\nexpected: %v\n     got: %v", tt.expected, actual)
		}
	}
}
