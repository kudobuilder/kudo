package init

import (
	"errors"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
)

// WatchKUDOUntilReady waits for the KUDO pod to become available.
//
// Returns true if it exists. If the timeout was reached and it could not find the pod, it returns false.
func WatchKUDOUntilReady(client kubernetes.Interface, opts Options, timeout int64) bool {
	deadlineChan := time.NewTimer(time.Duration(timeout) * time.Second).C
	checkPodTicker := time.NewTicker(500 * time.Millisecond)
	doneChan := make(chan bool)

	defer checkPodTicker.Stop()

	go func() {
		for range checkPodTicker.C {
			image, err := getKUDOPodImage(client.CoreV1(), opts.Namespace)
			if err == nil && image == opts.Image {
				doneChan <- true
				break
			}
		}
	}()

	for {
		select {
		case <-deadlineChan:
			return false
		case <-doneChan:
			return true
		}
	}
}

// getKUDOPodImage fetches the image of KUDO pod running in the given namespace.
func getKUDOPodImage(client corev1.PodsGetter, namespace string) (string, error) {
	selector := managerLabels().AsSelector()
	pod, err := getFirstRunningPod(client, namespace, selector)
	if err != nil {
		return "", err
	}
	for _, c := range pod.Spec.Containers {
		if c.Name == "manager" {
			return c.Image, nil
		}
	}
	return "", errors.New("could not find a KUDO pod")
}

func getFirstRunningPod(client corev1.PodsGetter, namespace string, selector labels.Selector) (*v1.Pod, error) {
	options := metav1.ListOptions{LabelSelector: selector.String()}
	pods, err := client.Pods(namespace).List(options)
	if err != nil {
		return nil, err
	}
	if len(pods.Items) < 1 {
		return nil, errors.New("could not find KUDO manager")
	}
	for _, p := range pods.Items {
		if kube.IsPodReady(&p) {
			return &p, nil
		}
	}
	return nil, errors.New("could not find a ready KUDO pod")
}
