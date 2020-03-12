package resource

import (
	"fmt"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ObjectKeyFromObject method wraps client.ObjectKeyFromObject method by additionally checking if passed object is
// a cluster-scoped resource (e.g. CustomResourceDefinition, ClusterRole etc.) and removing the namespace from the
// key since cluster-scoped resources are not namespaced.
func ObjectKeyFromObject(r runtime.Object, di discovery.CachedDiscoveryInterface) (client.ObjectKey, error) {
	key, err := client.ObjectKeyFromObject(r)
	if err != nil {
		return client.ObjectKey{}, fmt.Errorf("failed to get an object key from object %v: %v", r.GetObjectKind(), err)
	}

	// if the resource is cluster-scoped we need to clear then namespace from the key
	isNamespaced, err := IsNamespacedObject(r, di)
	if err != nil {
		return client.ObjectKey{}, fmt.Errorf("failed to determine if the resource %v is cluster-scoped: %v", r.GetObjectKind(), err)
	}

	if !isNamespaced {
		key.Namespace = ""
	}
	return key, nil
}

func IsNamespacedObject(r runtime.Object, di discovery.CachedDiscoveryInterface) (bool, error) {
	gvk := r.GetObjectKind().GroupVersionKind()
	return isNamespaced(gvk, di)
}

// isNamespaced method return true if given runtime.Object is a namespaced (not cluster-scoped) resource. It uses the
// discovery client to fetch all API resources (with Groups and Versions), searches for a resource with the passed GVK
// and returns true if it's namespaced. Method returns an error if passed GVK wasn't found in the discovered resource list.
func isNamespaced(gvk schema.GroupVersionKind, di discovery.CachedDiscoveryInterface) (bool, error) {
	// Fetch namespaced API resources

	apiResource, err := getAPIResource(gvk, di)
	if err != nil {
		return false, err
	}
	if apiResource == nil {
		log.Printf("Failed to get APIResource for %v, retry with invalidated cache.", gvk)
		di.Invalidate()
		apiResource, err = getAPIResource(gvk, di)
	}
	if err != nil {
		return false, err
	}
	if apiResource == nil {
		return false, fmt.Errorf("a resource with GVK %v seems to be missing in API resource list", gvk)
	}
	return apiResource.Namespaced, nil
}

// getAPIResource returns a specific APIResource from the DiscoveryInterface or nil if no resource was found.
// As the CachedDI may contain stale data, it can return nil even if the resource actually exists, in that
// case it is advised to invalidate the DI cache and retry the query
func getAPIResource(gvk schema.GroupVersionKind, di discovery.CachedDiscoveryInterface) (*metav1.APIResource, error) {
	_, apiResources, err := di.ServerGroupsAndResources()
	if err != nil {
		if err == memory.ErrCacheNotFound {
			return nil, nil
		}
		return nil, err
	}

	for _, rr := range apiResources {
		gv, err := schema.ParseGroupVersion(rr.GroupVersion)
		if err != nil {
			continue
		}
		for _, r := range rr.APIResources {
			r := r
			if gvk == gv.WithKind(r.Kind) {
				return &r, nil
			}
		}
	}
	return nil, nil
}
