package plan

import (
	"log"
	"os"
	"os/user"
	"path/filepath"
)

const (
	defaultConfigPath = ".kube/config"
)

var (
	instance   string
	kubeConfig string
	namespace  string
)

// mustKubeConfig checks if the kubeconfig file exists.
func mustKubeConfig() {
	// if kubeConfig is not specified, search for the default kubeconfig file under the $HOME/.kube/config.
	if len(kubeConfig) == 0 {
		usr, err := user.Current()
		if err != nil {
			log.Printf("Error: failed to determine user's home dir: %v", err)
		}
		kubeConfig = filepath.Join(usr.HomeDir, defaultConfigPath)
	}

	_, err := os.Stat(kubeConfig)
	if err != nil && os.IsNotExist(err) {
		log.Fatalf("Error: failed to find kubeconfig file (%v): %v", kubeConfig, err)
	}
}
