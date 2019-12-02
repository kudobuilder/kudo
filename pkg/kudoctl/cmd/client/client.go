package client

import (
	"fmt"
	"os"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	kudoinit "github.com/kudobuilder/kudo/pkg/kudoctl/cmd/init"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

// GetClient is a helper function that takes the Settings struct and returns a new KUDO Client
func GetValidatedClient(s *env.Settings) (*kudo.Client, error) {

	kubeClient, err := kube.GetKubeClient(s.KubeConfig)
	if err != nil {
		return nil, clog.Errorf("could not get Kubernetes client: %s", err)
	}

	err = kudoinit.CRDs().ValidateInstallation(kubeClient)
	if err != nil {
		// see above
		if os.IsTimeout(err) {
			return nil, err
		}
		clog.V(0).Printf("KUDO CRDs are not set up correctly. Do you need to run kudo init?")
		if s.Validate {
			return nil, fmt.Errorf("CRDs invalid: %v", err)
		}
	}

	return kudo.NewClient(s.KubeConfig, s.RequestTimeout)
}
