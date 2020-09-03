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

func IsKnownObjectType(r runtime.Object, di discovery.CachedDiscoveryInterface) (bool, error) {
	gvk := r.GetObjectKind().GroupVersionKind()
	return isKnownType(gvk, di)
}

// isNamespaced method return true if given runtime.Object is a namespaced (not cluster-scoped) resource. It uses the
// discovery client to fetch all API resources (with Groups and Versions), searches for a resource with the passed GVK
// and returns true if it's namespaced. Method returns an error if passed GVK wasn't found in the discovered resource list.
func isNamespaced(gvk schema.GroupVersionKind, di discovery.CachedDiscoveryInterface) (bool, error) {
	apiResource, err := getUncachedAPIResource(gvk, di)
	if err != nil {
		return false, err
	}
	if apiResource != nil {
		return apiResource.Namespaced, nil
	}
	return false, fmt.Errorf("a resource with GVK %v seems to be missing in API resource list", gvk)
}

func isKnownType(gvk schema.GroupVersionKind, di discovery.CachedDiscoveryInterface) (bool, error) {
	apiResource, err := getUncachedAPIResource(gvk, di)
	if err != nil {
		return false, err
	}
	if apiResource != nil {
		return true, nil
	}
	return false, nil
}

// getUncachedAPIResource tries to invalidate the cache and requery the discovery interface to make sure no stale data is returned
func getUncachedAPIResource(gvk schema.GroupVersionKind, di discovery.CachedDiscoveryInterface) (*metav1.APIResource, error) {
	// First try, this may return nil because of the cache
	apiResource, err := getAPIResource(gvk, di)
	if err != nil {
		return nil, err
	}
	if apiResource != nil {
		return apiResource, nil
	}

	// Second try, now with invalidated cache. If we still get nil, we know it's not there.
	log.Printf("Failed to get APIResource for %v, retry with invalidated cache.", gvk)
	di.Invalidate()
	apiResource, err = getAPIResource(gvk, di)
	if err != nil {
		return nil, err
	}
	if apiResource != nil {
		return apiResource, nil
	}

	return nil, nil
}

// getAPIResource returns a specific APIResource from the DiscoveryInterface or nil if no resource was found.
// As the CachedDiscoverInterface may contain stale data, it can return nil even if the resource actually exists, in that
// case it is advised to invalidate the DI cache and retry the query
// Additionally, this method may return false positives, i.e. an API resource that was already deleted from the api
// server. If no false positive results is required, call di.Invalidate before calling this method
func getAPIResource(gvk schema.GroupVersionKind, di discovery.CachedDiscoveryInterface) (*metav1.APIResource, error) {
	resList, err := di.ServerResourcesForGroupVersion(gvk.GroupVersion().String())

	if err != nil || resList == nil {
		if err == memory.ErrCacheNotFound {
			return nil, nil
		}
		return nil, err
	}

	gv, err := schema.ParseGroupVersion(resList.GroupVersion)
	if err != nil {
		return nil, err
	}
	for _, r := range resList.APIResources {
		if gvk == gv.WithKind(r.Kind) {
			return &r, nil
		}
	}

	return nil, nil
}
