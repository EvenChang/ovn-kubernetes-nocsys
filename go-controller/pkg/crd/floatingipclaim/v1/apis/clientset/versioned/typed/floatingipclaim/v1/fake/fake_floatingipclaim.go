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

	floatingipclaimv1 "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/floatingipclaim/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeFloatingIPClaims implements FloatingIPClaimInterface
type FakeFloatingIPClaims struct {
	Fake *FakeK8sV1
}

var floatingipclaimsResource = schema.GroupVersionResource{Group: "k8s.ovn.org", Version: "v1", Resource: "floatingipclaims"}

var floatingipclaimsKind = schema.GroupVersionKind{Group: "k8s.ovn.org", Version: "v1", Kind: "FloatingIPClaim"}

// Get takes name of the floatingIPClaim, and returns the corresponding floatingIPClaim object, and an error if there is any.
func (c *FakeFloatingIPClaims) Get(ctx context.Context, name string, options v1.GetOptions) (result *floatingipclaimv1.FloatingIPClaim, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(floatingipclaimsResource, name), &floatingipclaimv1.FloatingIPClaim{})
	if obj == nil {
		return nil, err
	}
	return obj.(*floatingipclaimv1.FloatingIPClaim), err
}

// List takes label and field selectors, and returns the list of FloatingIPClaims that match those selectors.
func (c *FakeFloatingIPClaims) List(ctx context.Context, opts v1.ListOptions) (result *floatingipclaimv1.FloatingIPClaimList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(floatingipclaimsResource, floatingipclaimsKind, opts), &floatingipclaimv1.FloatingIPClaimList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &floatingipclaimv1.FloatingIPClaimList{ListMeta: obj.(*floatingipclaimv1.FloatingIPClaimList).ListMeta}
	for _, item := range obj.(*floatingipclaimv1.FloatingIPClaimList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested floatingIPClaims.
func (c *FakeFloatingIPClaims) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(floatingipclaimsResource, opts))
}

// Create takes the representation of a floatingIPClaim and creates it.  Returns the server's representation of the floatingIPClaim, and an error, if there is any.
func (c *FakeFloatingIPClaims) Create(ctx context.Context, floatingIPClaim *floatingipclaimv1.FloatingIPClaim, opts v1.CreateOptions) (result *floatingipclaimv1.FloatingIPClaim, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(floatingipclaimsResource, floatingIPClaim), &floatingipclaimv1.FloatingIPClaim{})
	if obj == nil {
		return nil, err
	}
	return obj.(*floatingipclaimv1.FloatingIPClaim), err
}

// Update takes the representation of a floatingIPClaim and updates it. Returns the server's representation of the floatingIPClaim, and an error, if there is any.
func (c *FakeFloatingIPClaims) Update(ctx context.Context, floatingIPClaim *floatingipclaimv1.FloatingIPClaim, opts v1.UpdateOptions) (result *floatingipclaimv1.FloatingIPClaim, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(floatingipclaimsResource, floatingIPClaim), &floatingipclaimv1.FloatingIPClaim{})
	if obj == nil {
		return nil, err
	}
	return obj.(*floatingipclaimv1.FloatingIPClaim), err
}

// Delete takes name of the floatingIPClaim and deletes it. Returns an error if one occurs.
func (c *FakeFloatingIPClaims) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(floatingipclaimsResource, name), &floatingipclaimv1.FloatingIPClaim{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFloatingIPClaims) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(floatingipclaimsResource, listOpts)

	_, err := c.Fake.Invokes(action, &floatingipclaimv1.FloatingIPClaimList{})
	return err
}

// Patch applies the patch and returns the patched floatingIPClaim.
func (c *FakeFloatingIPClaims) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *floatingipclaimv1.FloatingIPClaim, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(floatingipclaimsResource, name, pt, data, subresources...), &floatingipclaimv1.FloatingIPClaim{})
	if obj == nil {
		return nil, err
	}
	return obj.(*floatingipclaimv1.FloatingIPClaim), err
}
