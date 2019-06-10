package check

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/pkg/errors"
)

const (
	defaultKubeConfigPath = ".kube/config"
)

// ValidateKubeConfigPath checks if the kubeconfig file exists.
func ValidateKubeConfigPath(path string) error {
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return errors.Wrap(err, "failed to find kubeconfig file")
	} else if stat.IsDir() {
		return errors.Wrap(fmt.Errorf("%v is a directory", path), "getting config failed")
	}
	return nil
}

// KubeConfigLocationOrDefault returns provided kubeconfig location or default if empty
func KubeConfigLocationOrDefault(location string) (string, error) {
	// if location is not specified, set the default kubeconfig file to $HOME/.kube/config.
	if len(location) == 0 {
		usr, err := user.Current()
		if err != nil {
			return "", errors.Wrap(err, "failed to determine user's home dir")
		}
		return filepath.Join(usr.HomeDir, defaultKubeConfigPath), nil
	}
	return location, nil
}
