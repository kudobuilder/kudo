package setup

import (
	"context"
	"time"

	"github.com/kudobuilder/kudo/pkg/engine/health"

	"k8s.io/apimachinery/pkg/util/wait"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
)

// WatchKUDOUntilReady waits for the KUDO pod to become available.
//
// Returns no error if it exists. If the timeout was reached and it could not find the pod, it returns error.
func WatchKUDOUntilReady(client kubernetes.Interface, opts kudoinit.Options, timeout int64) error {
	return wait.PollImmediate(500*time.Millisecond, time.Duration(timeout)*time.Second,
		func() (bool, error) { return verifyKudoStatefulset(client.AppsV1(), opts.Namespace) })
}

func verifyKudoStatefulset(client appsv1.StatefulSetsGetter, namespace string) (bool, error) {
	ss, err := client.StatefulSets(namespace).Get(context.TODO(), kudoinit.DefaultManagerName, metav1.GetOptions{})
	if err != nil || ss == nil {
		return false, err
	}
	err = health.IsHealthy(ss)
	if err != nil {
		return false, nil
	}
	return true, nil
}
