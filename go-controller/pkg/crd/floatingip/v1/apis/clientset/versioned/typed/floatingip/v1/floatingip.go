/*
Copyright The Kubernetes Authors.

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

package v1

import (
	"context"
	"time"

	v1 "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingip/v1"
	scheme "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingip/v1/apis/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// FloatingIPsGetter has a method to return a FloatingIPInterface.
// A group's client should implement this interface.
type FloatingIPsGetter interface {
	FloatingIPs() FloatingIPInterface
}

// FloatingIPInterface has methods to work with FloatingIP resources.
type FloatingIPInterface interface {
	Create(ctx context.Context, floatingIP *v1.FloatingIP, opts metav1.CreateOptions) (*v1.FloatingIP, error)
	Update(ctx context.Context, floatingIP *v1.FloatingIP, opts metav1.UpdateOptions) (*v1.FloatingIP, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.FloatingIP, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.FloatingIPList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.FloatingIP, err error)
	FloatingIPExpansion
}

// floatingIPs implements FloatingIPInterface
type floatingIPs struct {
	client rest.Interface
}

// newFloatingIPs returns a FloatingIPs
func newFloatingIPs(c *K8sV1Client) *floatingIPs {
	return &floatingIPs{
		client: c.RESTClient(),
	}
}

// Get takes name of the floatingIP, and returns the corresponding floatingIP object, and an error if there is any.
func (c *floatingIPs) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.FloatingIP, err error) {
	result = &v1.FloatingIP{}
	err = c.client.Get().
		Resource("floatingips").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FloatingIPs that match those selectors.
func (c *floatingIPs) List(ctx context.Context, opts metav1.ListOptions) (result *v1.FloatingIPList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.FloatingIPList{}
	err = c.client.Get().
		Resource("floatingips").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested floatingIPs.
func (c *floatingIPs) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("floatingips").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a floatingIP and creates it.  Returns the server's representation of the floatingIP, and an error, if there is any.
func (c *floatingIPs) Create(ctx context.Context, floatingIP *v1.FloatingIP, opts metav1.CreateOptions) (result *v1.FloatingIP, err error) {
	result = &v1.FloatingIP{}
	err = c.client.Post().
		Resource("floatingips").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(floatingIP).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a floatingIP and updates it. Returns the server's representation of the floatingIP, and an error, if there is any.
func (c *floatingIPs) Update(ctx context.Context, floatingIP *v1.FloatingIP, opts metav1.UpdateOptions) (result *v1.FloatingIP, err error) {
	result = &v1.FloatingIP{}
	err = c.client.Put().
		Resource("floatingips").
		Name(floatingIP.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(floatingIP).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the floatingIP and deletes it. Returns an error if one occurs.
func (c *floatingIPs) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Resource("floatingips").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *floatingIPs) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("floatingips").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched floatingIP.
func (c *floatingIPs) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.FloatingIP, err error) {
	result = &v1.FloatingIP{}
	err = c.client.Patch(pt).
		Resource("floatingips").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
