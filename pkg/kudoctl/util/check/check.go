package check

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
	"github.com/pkg/errors"
)

const (
	defaultKUDORepoPath   = ".kudo/repository"
	defaultKubeConfigPath = ".kube/config"
)

// ValidateKubeConfigPath checks if the kubeconfig file exists.
func ValidateKubeConfigPath() error {
	path, err := getKubeConfigLocation()
	if err != nil {
		return err
	}

	vars.KubeConfigPath = path
	if _, err := os.Stat(vars.KubeConfigPath); os.IsNotExist(err) {
		return errors.Wrap(err, "failed to find kubeconfig file")
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

// RepoPath checks if the repo path folder exists.
func RepoPath() error {
	// Option to set a repo path within the environment
	// overrides passed repo path parameter
	repoPathEnv := os.Getenv("KUDO_REPO_PATH")
	if repoPathEnv != "" {
		vars.RepoPath = repoPathEnv
	} else {
		usr, err := user.Current()
		if err != nil {
			return errors.Wrap(err, "failed to determine user's home dir")
		}
		vars.RepoPath = filepath.Join(usr.HomeDir, defaultKUDORepoPath)
	}

	if _, err := os.Stat(vars.RepoPath); err != nil && os.IsNotExist(err) {
		return errors.Wrap(err, "repo path does not exist")
	}
	return nil
}
