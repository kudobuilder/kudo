package diagnostics

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

// TODO: method copied from pkg/kudoctl/util/kudo. Probably should be made exported
// unset GVK on kubernetes objects breaks printing and describing, this method is a fix
func setGVKFromScheme(object runtime.Object) error {
	gvks, unversioned, err := scheme.Scheme.ObjectKinds(object)
	if err != nil {
		return err
	}
	if len(gvks) == 0 {
		return fmt.Errorf("no ObjectKinds available for %T", object)
	}
	if !unversioned {
		object.GetObjectKind().SetGroupVersionKind(gvks[0])
	}
	return nil
}
