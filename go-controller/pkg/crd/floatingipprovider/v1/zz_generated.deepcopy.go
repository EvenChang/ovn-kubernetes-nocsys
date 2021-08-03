// +build !ignore_autogenerated

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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FloatingIPProvider) DeepCopyInto(out *FloatingIPProvider) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FloatingIPProvider.
func (in *FloatingIPProvider) DeepCopy() *FloatingIPProvider {
	if in == nil {
		return nil
	}
	out := new(FloatingIPProvider)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *FloatingIPProvider) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FloatingIPProviderList) DeepCopyInto(out *FloatingIPProviderList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]FloatingIPProvider, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FloatingIPProviderList.
func (in *FloatingIPProviderList) DeepCopy() *FloatingIPProviderList {
	if in == nil {
		return nil
	}
	out := new(FloatingIPProviderList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *FloatingIPProviderList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FloatingIPProviderSpec) DeepCopyInto(out *FloatingIPProviderSpec) {
	*out = *in
	if in.FloatingIPs != nil {
		in, out := &in.FloatingIPs, &out.FloatingIPs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.VpcSelector.DeepCopyInto(&out.VpcSelector)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FloatingIPProviderSpec.
func (in *FloatingIPProviderSpec) DeepCopy() *FloatingIPProviderSpec {
	if in == nil {
		return nil
	}
	out := new(FloatingIPProviderSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FloatingIPProviderStatus) DeepCopyInto(out *FloatingIPProviderStatus) {
	*out = *in
	if in.FloatingIPClaims != nil {
		in, out := &in.FloatingIPClaims, &out.FloatingIPClaims
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FloatingIPProviderStatus.
func (in *FloatingIPProviderStatus) DeepCopy() *FloatingIPProviderStatus {
	if in == nil {
		return nil
	}
	out := new(FloatingIPProviderStatus)
	in.DeepCopyInto(out)
	return out
}
