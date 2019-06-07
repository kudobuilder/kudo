package check

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
	"github.com/pkg/errors"
)

const (
	defaultKubeConfigPath = ".kube/config"
)

// ValidateKubeConfigPath checks if the kubeconfig file exists.
func ValidateKubeConfigPath() error {
	path, err := getKubeConfigLocation()
	if err != nil {
		return err
	}

	vars.KubeConfigPath = path
	stat, err := os.Stat(vars.KubeConfigPath)
	if os.IsNotExist(err) {
		return errors.Wrap(err, "failed to find kubeconfig file")
	} else if stat.IsDir() {
		return errors.Wrap(fmt.Errorf("%v is a directory", vars.KubeConfigPath), "getting config failed")
	}
	return nil
}

func getKubeConfigLocation() (string, error) {
	// if vars.KubeConfigPath is not specified, search for the default kubeconfig file under the $HOME/.kube/config.
	if len(vars.KubeConfigPath) == 0 {
		usr, err := user.Current()
		if err != nil {
			return "", errors.Wrap(err, "failed to determine user's home dir")
		}
		return filepath.Join(usr.HomeDir, defaultKubeConfigPath), nil
	}
	return vars.KubeConfigPath, nil
}
