package check

import (
	"os/user"
	"path/filepath"
	"testing"
)

func TestKubeConfigPath(t *testing.T) {
	// first we test that the vars.KubeConfigPath is propagated correctly when resolving the path
	kubeConfigPath := "/tmp/;"
	location, err := KubeConfigLocationOrDefault(kubeConfigPath)
	if err != nil {
		t.Errorf("expected kubeconfig path '%v' to be propagated from vars, got error instead %v", kubeConfigPath, err)
	}
	if location != kubeConfigPath {
		t.Errorf("expected kubeconfig path '%v' to be propagated from vars, kubeconfig path instead resolved as %v", kubeConfigPath, location)
	}

	// then we test that default is used when no path is provided in vars
	kubeConfigPath = ""
	usr, _ := user.Current()
	expectedPath := filepath.Join(usr.HomeDir, defaultKubeConfigPath)
	location, err = KubeConfigLocationOrDefault(kubeConfigPath)
	if err != nil {
		t.Errorf("expected kubeconfig path '%v', got error instead %v", expectedPath, err)
	}
	if location != expectedPath {
		t.Errorf("expected kubeconfig path '%v', kubeconfig path instead resolved as %v", expectedPath, location)
	}
}
