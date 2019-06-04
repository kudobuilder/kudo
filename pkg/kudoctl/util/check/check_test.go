package check

import (
	"fmt"
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

func TestRepoPath(t *testing.T) {
	usr, err := user.Current()
	if err != nil {
		t.Error("failed to determine user's home dir")
	}
	usrDir := filepath.Join(usr.HomeDir, ".kudo/repository")

	tests := []struct {
		err string
	}{
		{fmt.Sprintf("repo path does not exist: stat %s: no such file or directory", usrDir)},
	}

	for _, tt := range tests {
		// Just interested in errors
		err := RepoPath()
		if err != nil {
			if err.Error() != tt.err {
				t.Errorf("non existing test:\nexpected: %v\n     got: %v", tt.err, err.Error())
			}
		}
	}
}
