package migration

import (
	"context"
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
)

func ForEachNamespace(client *kube.Client, f func(ns string) error) error {
	nsList, err := client.KubeClient.CoreV1().Namespaces().List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to fetch namespaces: %v", err)
	}
	for _, ns := range nsList.Items {
		if err := f(ns.Name); err != nil {
			return fmt.Errorf("failed to run function for namespace %q: %v", ns.Name, err)
		}
	}
	return nil
}

func ForEachOperatorVersion(client *kube.Client, ns string, f func(ov *kudoapi.OperatorVersion) error) error {
	ovList, err := client.KudoClient.KudoV1beta1().OperatorVersions(ns).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to fetch OperatorVersions from namespace %q: %v", ns, err)
	}
	for _, ov := range ovList.Items {
		ov := ov
		if err := f(&ov); err != nil {
			return fmt.Errorf("failed to run function for OperatorVersion \"%s/%s\": %v", ov.Namespace, ov.Name, err)
		}
	}
	return nil
}

func ForEachInstance(client *kube.Client, ns string, f func(ov *kudoapi.Instance) error) error {
	iList, err := client.KudoClient.KudoV1beta1().Instances(ns).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to fetch Instances from namespace %q: %v", ns, err)
	}
	for _, instance := range iList.Items {
		instance := instance
		if err := f(&instance); err != nil {
			return fmt.Errorf("failed to run function for Instance \"%s/%s\": %v", instance.Namespace, instance.Name, err)
		}
	}
	return nil
}
