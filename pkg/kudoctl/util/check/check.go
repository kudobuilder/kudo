package check

import (
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
	"github.com/pkg/errors"
	"os"
	"os/user"
	"path/filepath"
)

const (
	defaultKubeConfigPath       = ".kube/config"
	defaultGithubCredentialPath = ".git-credentials"
)

// KubeConfig checks if the kubeconfig file exists.
func KubeConfigPath() error {
	// if kubeConfig is not specified, search for the default kubeconfig file under the $HOME/.kube/config.
	if len(vars.KubeConfigPath) == 0 {
		usr, err := user.Current()
		if err != nil {
			return errors.Wrap(err, "failed to determine user's home dir")
		}
		vars.KubeConfigPath = filepath.Join(usr.HomeDir, defaultKubeConfigPath)
	}

	_, err := os.Stat(vars.KubeConfigPath)
	if err != nil && os.IsNotExist(err) {
		return errors.Wrap(err, "failed to find kubeconfig file")
	}
	return nil
}

// GithubCredentials checks if the credential file exists.
func GithubCredentials() error {
	// if credentials are not specified, search for the default credential file under the $HOME/.git-credentials.
	if len(vars.GithubCredentialPath) == 0 {
		usr, err := user.Current()
		if err != nil {
			return errors.Wrap(err, "failed to determine user's home dir")
		}
		vars.GithubCredentialPath = filepath.Join(usr.HomeDir, defaultGithubCredentialPath)
	}

	_, err := os.Stat(vars.GithubCredentialPath)
	if err != nil && os.IsNotExist(err) {
		return errors.Wrap(err, "failed to find github credential file")
	}
	return nil
}
