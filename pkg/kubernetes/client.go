package kubernetes

import (
	"fmt"

	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func GetDiscoveryClient(mgr manager.Manager) (*discovery.DiscoveryClient, error) {
	// use manager rest config to create a discovery client
	dc, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil || dc == nil {
		return nil, fmt.Errorf("failed to create a discovery client: %v", err)
	}
	return dc, nil
}
