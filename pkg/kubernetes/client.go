package kubernetes

import (
	"context"
	"fmt"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
)

func GetDiscoveryClient(mgr manager.Manager) (*discovery.DiscoveryClient, error) {
	// use manager rest config to create a discovery client
	dc, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil || dc == nil {
		return nil, fmt.Errorf("failed to create a discovery client: %v", err)
	}
	return dc, nil
}

// DeleteAndWait deletes the given runtime object and waits until it is fully deleted
func DeleteAndWait(c client.Client, obj runtime.Object, options ...client.DeleteOption) error {
	err := c.Delete(context.TODO(), obj, options...)

	if err != nil {
		key := ObjectKey(obj)

		if kerrors.IsNotFound(err) {
			// Obj is already deleted, we can return directly
			clog.V(4).Printf("Deleting obj %s/%s is already NotFound, return now", key.Namespace, key.Name)
			return nil
		}
		return fmt.Errorf("failed to delete %s/%s: %v", key.Namespace, key.Name, err)
	}

	return WaitForDelete(c, obj)
}

// WaitForDelete waits for the provided runtime object to be deleted from cluster
func WaitForDelete(c client.Client, obj runtime.Object) error {
	key := ObjectKey(obj)
	clog.V(4).Printf("Waiting for obj %s/%s to be finally deleted", key.Namespace, key.Name)

	// Wait for resources to be deleted.
	return wait.PollImmediate(250*time.Millisecond, 30*time.Second, func() (done bool, err error) {
		err = c.Get(context.TODO(), key, obj.DeepCopyObject())
		clog.V(6).Printf("Fetched %s/%s to wait for delete: %v", key.Namespace, key.Name, err)

		if err != nil && kerrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}

// ObjectKey returns an instantiated ObjectKey for the provided object.
func ObjectKey(obj runtime.Object) client.ObjectKey {
	m, _ := meta.Accessor(obj)
	return client.ObjectKey{
		Name:      m.GetName(),
		Namespace: m.GetNamespace(),
	}
}
