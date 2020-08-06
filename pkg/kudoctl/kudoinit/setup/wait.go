package setup

import (
	"context"
	"errors"
	"time"

	"github.com/kudobuilder/kudo/pkg/engine/health"

	"k8s.io/apimachinery/pkg/util/wait"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/manager"
)

// WatchKUDOUntilReady waits for the KUDO pod to become available.
//
// Returns true if it exists. If the timeout was reached and it could not find the pod, it returns false.
func WatchKUDOUntilReady(client kubernetes.Interface, opts kudoinit.Options, timeout int64) error {
	return wait.PollImmediate(500*time.Millisecond, time.Duration(timeout)*time.Second,
		func() (bool, error) { return verifyKudoDeployment(client.CoreV1(), opts.Namespace) })
}

func verifyKudoDeployment(client corev1.PodsGetter, namespace string) (bool, error) {
	ready, err := isKUDOPodReady(client, namespace)
	if err == nil && ready {
		return true, nil
	}
	return false, nil
}

// isKUDOPodReady fetches the KUDO pod running in the given namespace and returns true if Ready Condition has a status of true
func isKUDOPodReady(client corev1.PodsGetter, namespace string) (bool, error) {
	selector := manager.GenerateLabels().AsSelector()
	pod, err := getFirstReadyPod(client, namespace, selector)
	if err != nil || pod == nil {
		return false, err
	}
	return true, nil
}

func getFirstReadyPod(client corev1.PodsGetter, namespace string, selector labels.Selector) (*v1.Pod, error) { //nolint:interfacer
	options := metav1.ListOptions{LabelSelector: selector.String()}
	pods, err := client.Pods(namespace).List(context.TODO(), options)
	if err != nil {
		return nil, err
	}
	if len(pods.Items) < 1 {
		return nil, errors.New("could not find KUDO manager")
	}
	for _, p := range pods.Items {
		p := p

		if health.IsHealthy(&p) == nil {
			return &p, nil
		}
	}
	return nil, errors.New("could not find a ready KUDO pod")
}
