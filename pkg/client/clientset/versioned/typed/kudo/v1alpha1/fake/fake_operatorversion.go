/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeOperatorVersions implements OperatorVersionInterface
type FakeOperatorVersions struct {
	Fake *FakeKudoV1alpha1
	ns   string
}

var operatorversionsResource = schema.GroupVersionResource{Group: "kudo.dev", Version: "v1alpha1", Resource: "operatorversions"}

var operatorversionsKind = schema.GroupVersionKind{Group: "kudo.dev", Version: "v1alpha1", Kind: "OperatorVersion"}

// Get takes name of the operatorVersion, and returns the corresponding operatorVersion object, and an error if there is any.
func (c *FakeOperatorVersions) Get(name string, options v1.GetOptions) (result *v1alpha1.OperatorVersion, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(operatorversionsResource, c.ns, name), &v1alpha1.OperatorVersion{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.OperatorVersion), err
}

// List takes label and field selectors, and returns the list of OperatorVersions that match those selectors.
func (c *FakeOperatorVersions) List(opts v1.ListOptions) (result *v1alpha1.OperatorVersionList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(operatorversionsResource, operatorversionsKind, c.ns, opts), &v1alpha1.OperatorVersionList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.OperatorVersionList{ListMeta: obj.(*v1alpha1.OperatorVersionList).ListMeta}
	for _, item := range obj.(*v1alpha1.OperatorVersionList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested operatorVersions.
func (c *FakeOperatorVersions) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(operatorversionsResource, c.ns, opts))

}

// Create takes the representation of an operatorVersion and creates it.  Returns the server's representation of the operatorVersion, and an error, if there is any.
func (c *FakeOperatorVersions) Create(operatorVersion *v1alpha1.OperatorVersion) (result *v1alpha1.OperatorVersion, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(operatorversionsResource, c.ns, operatorVersion), &v1alpha1.OperatorVersion{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.OperatorVersion), err
}

// Update takes the representation of an operatorVersion and updates it. Returns the server's representation of the operatorVersion, and an error, if there is any.
func (c *FakeOperatorVersions) Update(operatorVersion *v1alpha1.OperatorVersion) (result *v1alpha1.OperatorVersion, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(operatorversionsResource, c.ns, operatorVersion), &v1alpha1.OperatorVersion{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.OperatorVersion), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeOperatorVersions) UpdateStatus(operatorVersion *v1alpha1.OperatorVersion) (*v1alpha1.OperatorVersion, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(operatorversionsResource, "status", c.ns, operatorVersion), &v1alpha1.OperatorVersion{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.OperatorVersion), err
}

// Delete takes name of the operatorVersion and deletes it. Returns an error if one occurs.
func (c *FakeOperatorVersions) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(operatorversionsResource, c.ns, name), &v1alpha1.OperatorVersion{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeOperatorVersions) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(operatorversionsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.OperatorVersionList{})
	return err
}

// Patch applies the patch and returns the patched operatorVersion.
func (c *FakeOperatorVersions) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.OperatorVersion, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(operatorversionsResource, c.ns, name, pt, data, subresources...), &v1alpha1.OperatorVersion{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.OperatorVersion), err
}
