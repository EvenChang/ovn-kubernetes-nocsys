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

package fake

import (
	"context"

	floatingipv1 "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingip/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeFloatingIPs implements FloatingIPInterface
type FakeFloatingIPs struct {
	Fake *FakeK8sV1
	ns   string
}

var floatingipsResource = schema.GroupVersionResource{Group: "k8s.ovn.org", Version: "v1", Resource: "floatingips"}

var floatingipsKind = schema.GroupVersionKind{Group: "k8s.ovn.org", Version: "v1", Kind: "FloatingIP"}

// Get takes name of the floatingIP, and returns the corresponding floatingIP object, and an error if there is any.
func (c *FakeFloatingIPs) Get(ctx context.Context, name string, options v1.GetOptions) (result *floatingipv1.FloatingIP, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(floatingipsResource, c.ns, name), &floatingipv1.FloatingIP{})

	if obj == nil {
		return nil, err
	}
	return obj.(*floatingipv1.FloatingIP), err
}

// List takes label and field selectors, and returns the list of FloatingIPs that match those selectors.
func (c *FakeFloatingIPs) List(ctx context.Context, opts v1.ListOptions) (result *floatingipv1.FloatingIPList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(floatingipsResource, floatingipsKind, c.ns, opts), &floatingipv1.FloatingIPList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &floatingipv1.FloatingIPList{ListMeta: obj.(*floatingipv1.FloatingIPList).ListMeta}
	for _, item := range obj.(*floatingipv1.FloatingIPList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested floatingIPs.
func (c *FakeFloatingIPs) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(floatingipsResource, c.ns, opts))

}

// Create takes the representation of a floatingIP and creates it.  Returns the server's representation of the floatingIP, and an error, if there is any.
func (c *FakeFloatingIPs) Create(ctx context.Context, floatingIP *floatingipv1.FloatingIP, opts v1.CreateOptions) (result *floatingipv1.FloatingIP, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(floatingipsResource, c.ns, floatingIP), &floatingipv1.FloatingIP{})

	if obj == nil {
		return nil, err
	}
	return obj.(*floatingipv1.FloatingIP), err
}

// Update takes the representation of a floatingIP and updates it. Returns the server's representation of the floatingIP, and an error, if there is any.
func (c *FakeFloatingIPs) Update(ctx context.Context, floatingIP *floatingipv1.FloatingIP, opts v1.UpdateOptions) (result *floatingipv1.FloatingIP, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(floatingipsResource, c.ns, floatingIP), &floatingipv1.FloatingIP{})

	if obj == nil {
		return nil, err
	}
	return obj.(*floatingipv1.FloatingIP), err
}

// Delete takes name of the floatingIP and deletes it. Returns an error if one occurs.
func (c *FakeFloatingIPs) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(floatingipsResource, c.ns, name), &floatingipv1.FloatingIP{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFloatingIPs) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(floatingipsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &floatingipv1.FloatingIPList{})
	return err
}

// Patch applies the patch and returns the patched floatingIP.
func (c *FakeFloatingIPs) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *floatingipv1.FloatingIP, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(floatingipsResource, c.ns, name, pt, data, subresources...), &floatingipv1.FloatingIP{})

	if obj == nil {
		return nil, err
	}
	return obj.(*floatingipv1.FloatingIP), err
}
