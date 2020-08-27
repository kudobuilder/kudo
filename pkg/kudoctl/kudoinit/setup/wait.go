package setup

import (
	"time"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
)

// WatchKUDOUntilReady waits for the KUDO installation to become available.
//
// Returns no error if it exists. If the timeout was reached and it could not find the pod, it returns error.
func WatchKUDOUntilReady(v kudoinit.InstallVerifier, client *kube.Client, timeout int64) error {
	return wait.PollImmediate(500*time.Millisecond, time.Duration(timeout)*time.Second,
		func() (bool, error) {
			return VerifyExistingInstallation(v, client, nil)
		},
	)
}
