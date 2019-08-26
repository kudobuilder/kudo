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

package v1alpha1

import (
	"time"

	v1alpha1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	scheme "github.com/kudobuilder/kudo/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// OperatorsGetter has a method to return a OperatorInterface.
// A group's client should implement this interface.
type OperatorsGetter interface {
	Operators(namespace string) OperatorInterface
}

// OperatorInterface has methods to work with Operator resources.
type OperatorInterface interface {
	Create(*v1alpha1.Operator) (*v1alpha1.Operator, error)
	Update(*v1alpha1.Operator) (*v1alpha1.Operator, error)
	UpdateStatus(*v1alpha1.Operator) (*v1alpha1.Operator, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.Operator, error)
	List(opts v1.ListOptions) (*v1alpha1.OperatorList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Operator, err error)
	OperatorExpansion
}

// operators implements OperatorInterface
type operators struct {
	client rest.Interface
	ns     string
}

// newOperators returns a Operators
func newOperators(c *KudoV1alpha1Client, namespace string) *operators {
	return &operators{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the operator, and returns the corresponding operator object, and an error if there is any.
func (c *operators) Get(name string, options v1.GetOptions) (result *v1alpha1.Operator, err error) {
	result = &v1alpha1.Operator{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("operators").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Operators that match those selectors.
func (c *operators) List(opts v1.ListOptions) (result *v1alpha1.OperatorList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.OperatorList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("operators").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested operators.
func (c *operators) Watch(opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("operators").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a operator and creates it.  Returns the server's representation of the operator, and an error, if there is any.
func (c *operators) Create(operator *v1alpha1.Operator) (result *v1alpha1.Operator, err error) {
	result = &v1alpha1.Operator{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("operators").
		Body(operator).
		Do().
		Into(result)
	return
}

// Update takes the representation of a operator and updates it. Returns the server's representation of the operator, and an error, if there is any.
func (c *operators) Update(operator *v1alpha1.Operator) (result *v1alpha1.Operator, err error) {
	result = &v1alpha1.Operator{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("operators").
		Name(operator.Name).
		Body(operator).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *operators) UpdateStatus(operator *v1alpha1.Operator) (result *v1alpha1.Operator, err error) {
	result = &v1alpha1.Operator{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("operators").
		Name(operator.Name).
		SubResource("status").
		Body(operator).
		Do().
		Into(result)
	return
}

// Delete takes name of the operator and deletes it. Returns an error if one occurs.
func (c *operators) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("operators").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *operators) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("operators").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched operator.
func (c *operators) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Operator, err error) {
	result = &v1alpha1.Operator{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("operators").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
