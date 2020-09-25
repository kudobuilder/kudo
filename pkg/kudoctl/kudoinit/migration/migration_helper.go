// This package contains helper functions that can be reused by all migrations
package migration

import (
	"context"
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
)

// ForEachNamespace calls the given function for all namespaces in the cluster
func ForEachNamespace(client *kube.Client, f func(ns string) error) error {
	l, err := client.KubeClient.CoreV1().Namespaces().List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to fetch namespaces: %v", err)
	}
	for _, ns := range l.Items {
		if err := f(ns.Name); err != nil {
			return fmt.Errorf("failed to run function for namespace %q: %v", ns.Name, err)
		}
	}
	return nil
}

// ForEachOperatorVersion calls the given function for all operatorversions in the given namespace
func ForEachOperatorVersion(client *kube.Client, ns string, f func(ov *kudoapi.OperatorVersion) error) error {
	l, err := client.KudoClient.KudoV1beta1().OperatorVersions(ns).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to fetch OperatorVersions from namespace %q: %v", ns, err)
	}
	for _, ov := range l.Items {
		ov := ov
		if err := f(&ov); err != nil {
			return fmt.Errorf("failed to run function for OperatorVersion \"%s/%s\": %v", ov.Namespace, ov.Name, err)
		}
	}
	return nil
}

// ForEachInstance calls the given function for all instances in the given namespace
func ForEachInstance(client *kube.Client, ns string, f func(ov *kudoapi.Instance) error) error {
	l, err := client.KudoClient.KudoV1beta1().Instances(ns).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to fetch Instances from namespace %q: %v", ns, err)
	}
	for _, i := range l.Items {
		i := i
		if err := f(&i); err != nil {
			return fmt.Errorf("failed to run function for Instance \"%s/%s\": %v", i.Namespace, i.Name, err)
		}
	}
	return nil
}
