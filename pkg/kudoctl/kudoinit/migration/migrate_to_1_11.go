package migration

import (
	"fmt"

	"github.com/thoas/go-funk"

	"github.com/prometheus/common/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kudobuilder/kudo/pkg/controller/instance"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/util/kudo"
)

var _ Migrater = &To1_11Migration{}

type To1_11Migration struct {
}

func To1_11() Migrater {
	return &To1_11Migration{}
}

func (m *To1_11Migration) String() string {
	return "to 1.11"
}

func (m *To1_11Migration) CanMigrate(client *kube.Client) error {
	listOptions := metav1.ListOptions{
		LabelSelector: kudo.InstanceLabel,
	}

	resLists, err := client.Discovery.ServerPreferredResources()
	if err != nil {
		return fmt.Errorf("failed to get resources: %v", err)
	}
	for _, rl := range resLists {
		for _, r := range rl.APIResources {

			if !funk.Contains(r.Verbs, "list") {
				continue
			}

			gv, err := schema.ParseGroupVersion(rl.GroupVersion)
			if err != nil {
				return fmt.Errorf("failed to parse group version %s: %v", rl.GroupVersion, err)
			}

			gvr := gv.WithResource(r.Name)

			items, err := client.DynamicClient.Resource(gvr).List(listOptions)
			if err != nil {
				return fmt.Errorf("failed to list resources for %v: %v", gvr, err)
			}
			for _, item := range items.Items {
				i := item
				u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&i)
				if err != nil {
					return fmt.Errorf("failed to convert res to unstructured %v: %v", i, err)
				}
				uObj := unstructured.Unstructured{Object: u}

				var oRef *metav1.OwnerReference
				for _, r := range uObj.GetOwnerReferences() {
					r := r
					if r.Kind == "Instance" && r.APIVersion == "kudo.dev/v1beta1" {
						oRef = &r
					}
				}

				if oRef == nil {
					log.Infof("Skipping %v - %v - %v, was not created by instance", uObj.GroupVersionKind(), uObj.GetNamespace(), uObj.GetName())
					continue
				}

				if _, ok := uObj.GetAnnotations()[instance.SnapshotAnnotation]; ok {
					log.Infof("Skipping %v - %v - %v, already has snapshot annotation", uObj.GroupVersionKind(), uObj.GetNamespace(), uObj.GetName())
					continue
				}
			}
		}
	}

	return nil
}

func (m *To1_11Migration) Migrate(client *kube.Client) error {
	return nil
}
