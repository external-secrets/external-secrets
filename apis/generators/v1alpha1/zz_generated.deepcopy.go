//go:build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/apis/meta/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ACRAccessToken) DeepCopyInto(out *ACRAccessToken) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ACRAccessToken.
func (in *ACRAccessToken) DeepCopy() *ACRAccessToken {
	if in == nil {
		return nil
	}
	out := new(ACRAccessToken)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ACRAccessToken) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ACRAccessTokenList) DeepCopyInto(out *ACRAccessTokenList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ACRAccessToken, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ACRAccessTokenList.
func (in *ACRAccessTokenList) DeepCopy() *ACRAccessTokenList {
	if in == nil {
		return nil
	}
	out := new(ACRAccessTokenList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ACRAccessTokenList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ACRAccessTokenSpec) DeepCopyInto(out *ACRAccessTokenSpec) {
	*out = *in
	in.Auth.DeepCopyInto(&out.Auth)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ACRAccessTokenSpec.
func (in *ACRAccessTokenSpec) DeepCopy() *ACRAccessTokenSpec {
	if in == nil {
		return nil
	}
	out := new(ACRAccessTokenSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ACRAuth) DeepCopyInto(out *ACRAuth) {
	*out = *in
	if in.ServicePrincipal != nil {
		in, out := &in.ServicePrincipal, &out.ServicePrincipal
		*out = new(AzureACRServicePrincipalAuth)
		(*in).DeepCopyInto(*out)
	}
	if in.ManagedIdentity != nil {
		in, out := &in.ManagedIdentity, &out.ManagedIdentity
		*out = new(AzureACRManagedIdentityAuth)
		**out = **in
	}
	if in.WorkloadIdentity != nil {
		in, out := &in.WorkloadIdentity, &out.WorkloadIdentity
		*out = new(AzureACRWorkloadIdentityAuth)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ACRAuth.
func (in *ACRAuth) DeepCopy() *ACRAuth {
	if in == nil {
		return nil
	}
	out := new(ACRAuth)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AWSAuth) DeepCopyInto(out *AWSAuth) {
	*out = *in
	if in.SecretRef != nil {
		in, out := &in.SecretRef, &out.SecretRef
		*out = new(AWSAuthSecretRef)
		(*in).DeepCopyInto(*out)
	}
	if in.JWTAuth != nil {
		in, out := &in.JWTAuth, &out.JWTAuth
		*out = new(AWSJWTAuth)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AWSAuth.
func (in *AWSAuth) DeepCopy() *AWSAuth {
	if in == nil {
		return nil
	}
	out := new(AWSAuth)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AWSAuthSecretRef) DeepCopyInto(out *AWSAuthSecretRef) {
	*out = *in
	in.AccessKeyID.DeepCopyInto(&out.AccessKeyID)
	in.SecretAccessKey.DeepCopyInto(&out.SecretAccessKey)
	if in.SessionToken != nil {
		in, out := &in.SessionToken, &out.SessionToken
		*out = new(v1.SecretKeySelector)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AWSAuthSecretRef.
func (in *AWSAuthSecretRef) DeepCopy() *AWSAuthSecretRef {
	if in == nil {
		return nil
	}
	out := new(AWSAuthSecretRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AWSIAMKey) DeepCopyInto(out *AWSIAMKey) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AWSIAMKey.
func (in *AWSIAMKey) DeepCopy() *AWSIAMKey {
	if in == nil {
		return nil
	}
	out := new(AWSIAMKey)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AWSIAMKey) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AWSIAMKeysList) DeepCopyInto(out *AWSIAMKeysList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]AWSIAMKey, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AWSIAMKeysList.
func (in *AWSIAMKeysList) DeepCopy() *AWSIAMKeysList {
	if in == nil {
		return nil
	}
	out := new(AWSIAMKeysList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AWSIAMKeysList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AWSJWTAuth) DeepCopyInto(out *AWSJWTAuth) {
	*out = *in
	if in.ServiceAccountRef != nil {
		in, out := &in.ServiceAccountRef, &out.ServiceAccountRef
		*out = new(v1.ServiceAccountSelector)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AWSJWTAuth.
func (in *AWSJWTAuth) DeepCopy() *AWSJWTAuth {
	if in == nil {
		return nil
	}
	out := new(AWSJWTAuth)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzureACRManagedIdentityAuth) DeepCopyInto(out *AzureACRManagedIdentityAuth) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzureACRManagedIdentityAuth.
func (in *AzureACRManagedIdentityAuth) DeepCopy() *AzureACRManagedIdentityAuth {
	if in == nil {
		return nil
	}
	out := new(AzureACRManagedIdentityAuth)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzureACRServicePrincipalAuth) DeepCopyInto(out *AzureACRServicePrincipalAuth) {
	*out = *in
	in.SecretRef.DeepCopyInto(&out.SecretRef)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzureACRServicePrincipalAuth.
func (in *AzureACRServicePrincipalAuth) DeepCopy() *AzureACRServicePrincipalAuth {
	if in == nil {
		return nil
	}
	out := new(AzureACRServicePrincipalAuth)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzureACRServicePrincipalAuthSecretRef) DeepCopyInto(out *AzureACRServicePrincipalAuthSecretRef) {
	*out = *in
	in.ClientID.DeepCopyInto(&out.ClientID)
	in.ClientSecret.DeepCopyInto(&out.ClientSecret)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzureACRServicePrincipalAuthSecretRef.
func (in *AzureACRServicePrincipalAuthSecretRef) DeepCopy() *AzureACRServicePrincipalAuthSecretRef {
	if in == nil {
		return nil
	}
	out := new(AzureACRServicePrincipalAuthSecretRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzureACRWorkloadIdentityAuth) DeepCopyInto(out *AzureACRWorkloadIdentityAuth) {
	*out = *in
	if in.ServiceAccountRef != nil {
		in, out := &in.ServiceAccountRef, &out.ServiceAccountRef
		*out = new(v1.ServiceAccountSelector)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzureACRWorkloadIdentityAuth.
func (in *AzureACRWorkloadIdentityAuth) DeepCopy() *AzureACRWorkloadIdentityAuth {
	if in == nil {
		return nil
	}
	out := new(AzureACRWorkloadIdentityAuth)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ControllerClassResource) DeepCopyInto(out *ControllerClassResource) {
	*out = *in
	out.Spec = in.Spec
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ControllerClassResource.
func (in *ControllerClassResource) DeepCopy() *ControllerClassResource {
	if in == nil {
		return nil
	}
	out := new(ControllerClassResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ECRAuthorizationToken) DeepCopyInto(out *ECRAuthorizationToken) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ECRAuthorizationToken.
func (in *ECRAuthorizationToken) DeepCopy() *ECRAuthorizationToken {
	if in == nil {
		return nil
	}
	out := new(ECRAuthorizationToken)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ECRAuthorizationToken) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ECRAuthorizationTokenList) DeepCopyInto(out *ECRAuthorizationTokenList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ECRAuthorizationToken, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ECRAuthorizationTokenList.
func (in *ECRAuthorizationTokenList) DeepCopy() *ECRAuthorizationTokenList {
	if in == nil {
		return nil
	}
	out := new(ECRAuthorizationTokenList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ECRAuthorizationTokenList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ECRAuthorizationTokenSpec) DeepCopyInto(out *ECRAuthorizationTokenSpec) {
	*out = *in
	in.Auth.DeepCopyInto(&out.Auth)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ECRAuthorizationTokenSpec.
func (in *ECRAuthorizationTokenSpec) DeepCopy() *ECRAuthorizationTokenSpec {
	if in == nil {
		return nil
	}
	out := new(ECRAuthorizationTokenSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Fake) DeepCopyInto(out *Fake) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Fake.
func (in *Fake) DeepCopy() *Fake {
	if in == nil {
		return nil
	}
	out := new(Fake)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Fake) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FakeList) DeepCopyInto(out *FakeList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Fake, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FakeList.
func (in *FakeList) DeepCopy() *FakeList {
	if in == nil {
		return nil
	}
	out := new(FakeList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *FakeList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FakeSpec) DeepCopyInto(out *FakeSpec) {
	*out = *in
	if in.Data != nil {
		in, out := &in.Data, &out.Data
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FakeSpec.
func (in *FakeSpec) DeepCopy() *FakeSpec {
	if in == nil {
		return nil
	}
	out := new(FakeSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GCPSMAuth) DeepCopyInto(out *GCPSMAuth) {
	*out = *in
	if in.SecretRef != nil {
		in, out := &in.SecretRef, &out.SecretRef
		*out = new(GCPSMAuthSecretRef)
		(*in).DeepCopyInto(*out)
	}
	if in.WorkloadIdentity != nil {
		in, out := &in.WorkloadIdentity, &out.WorkloadIdentity
		*out = new(GCPWorkloadIdentity)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GCPSMAuth.
func (in *GCPSMAuth) DeepCopy() *GCPSMAuth {
	if in == nil {
		return nil
	}
	out := new(GCPSMAuth)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GCPSMAuthSecretRef) DeepCopyInto(out *GCPSMAuthSecretRef) {
	*out = *in
	in.SecretAccessKey.DeepCopyInto(&out.SecretAccessKey)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GCPSMAuthSecretRef.
func (in *GCPSMAuthSecretRef) DeepCopy() *GCPSMAuthSecretRef {
	if in == nil {
		return nil
	}
	out := new(GCPSMAuthSecretRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GCPWorkloadIdentity) DeepCopyInto(out *GCPWorkloadIdentity) {
	*out = *in
	in.ServiceAccountRef.DeepCopyInto(&out.ServiceAccountRef)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GCPWorkloadIdentity.
func (in *GCPWorkloadIdentity) DeepCopy() *GCPWorkloadIdentity {
	if in == nil {
		return nil
	}
	out := new(GCPWorkloadIdentity)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GCRAccessToken) DeepCopyInto(out *GCRAccessToken) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GCRAccessToken.
func (in *GCRAccessToken) DeepCopy() *GCRAccessToken {
	if in == nil {
		return nil
	}
	out := new(GCRAccessToken)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *GCRAccessToken) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GCRAccessTokenList) DeepCopyInto(out *GCRAccessTokenList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]GCRAccessToken, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GCRAccessTokenList.
func (in *GCRAccessTokenList) DeepCopy() *GCRAccessTokenList {
	if in == nil {
		return nil
	}
	out := new(GCRAccessTokenList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *GCRAccessTokenList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GCRAccessTokenSpec) DeepCopyInto(out *GCRAccessTokenSpec) {
	*out = *in
	in.Auth.DeepCopyInto(&out.Auth)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GCRAccessTokenSpec.
func (in *GCRAccessTokenSpec) DeepCopy() *GCRAccessTokenSpec {
	if in == nil {
		return nil
	}
	out := new(GCRAccessTokenSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GithubAccessToken) DeepCopyInto(out *GithubAccessToken) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GithubAccessToken.
func (in *GithubAccessToken) DeepCopy() *GithubAccessToken {
	if in == nil {
		return nil
	}
	out := new(GithubAccessToken)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *GithubAccessToken) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GithubAccessTokenList) DeepCopyInto(out *GithubAccessTokenList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]GithubAccessToken, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GithubAccessTokenList.
func (in *GithubAccessTokenList) DeepCopy() *GithubAccessTokenList {
	if in == nil {
		return nil
	}
	out := new(GithubAccessTokenList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *GithubAccessTokenList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GithubAccessTokenSpec) DeepCopyInto(out *GithubAccessTokenSpec) {
	*out = *in
	in.Auth.DeepCopyInto(&out.Auth)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GithubAccessTokenSpec.
func (in *GithubAccessTokenSpec) DeepCopy() *GithubAccessTokenSpec {
	if in == nil {
		return nil
	}
	out := new(GithubAccessTokenSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GithubAuth) DeepCopyInto(out *GithubAuth) {
	*out = *in
	in.PrivateKey.DeepCopyInto(&out.PrivateKey)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GithubAuth.
func (in *GithubAuth) DeepCopy() *GithubAuth {
	if in == nil {
		return nil
	}
	out := new(GithubAuth)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GithubSecretRef) DeepCopyInto(out *GithubSecretRef) {
	*out = *in
	in.SecretRef.DeepCopyInto(&out.SecretRef)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GithubSecretRef.
func (in *GithubSecretRef) DeepCopy() *GithubSecretRef {
	if in == nil {
		return nil
	}
	out := new(GithubSecretRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IAMKeysSpec) DeepCopyInto(out *IAMKeysSpec) {
	*out = *in
	in.Auth.DeepCopyInto(&out.Auth)
	out.IAMRef = in.IAMRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IAMKeysSpec.
func (in *IAMKeysSpec) DeepCopy() *IAMKeysSpec {
	if in == nil {
		return nil
	}
	out := new(IAMKeysSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IAMRef) DeepCopyInto(out *IAMRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IAMRef.
func (in *IAMRef) DeepCopy() *IAMRef {
	if in == nil {
		return nil
	}
	out := new(IAMRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Password) DeepCopyInto(out *Password) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Password.
func (in *Password) DeepCopy() *Password {
	if in == nil {
		return nil
	}
	out := new(Password)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Password) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PasswordList) DeepCopyInto(out *PasswordList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Password, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PasswordList.
func (in *PasswordList) DeepCopy() *PasswordList {
	if in == nil {
		return nil
	}
	out := new(PasswordList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PasswordList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PasswordSpec) DeepCopyInto(out *PasswordSpec) {
	*out = *in
	if in.Digits != nil {
		in, out := &in.Digits, &out.Digits
		*out = new(int)
		**out = **in
	}
	if in.Symbols != nil {
		in, out := &in.Symbols, &out.Symbols
		*out = new(int)
		**out = **in
	}
	if in.SymbolCharacters != nil {
		in, out := &in.SymbolCharacters, &out.SymbolCharacters
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PasswordSpec.
func (in *PasswordSpec) DeepCopy() *PasswordSpec {
	if in == nil {
		return nil
	}
	out := new(PasswordSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretKeySelector) DeepCopyInto(out *SecretKeySelector) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretKeySelector.
func (in *SecretKeySelector) DeepCopy() *SecretKeySelector {
	if in == nil {
		return nil
	}
	out := new(SecretKeySelector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UUID) DeepCopyInto(out *UUID) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UUID.
func (in *UUID) DeepCopy() *UUID {
	if in == nil {
		return nil
	}
	out := new(UUID)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *UUID) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UUIDList) DeepCopyInto(out *UUIDList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Password, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UUIDList.
func (in *UUIDList) DeepCopy() *UUIDList {
	if in == nil {
		return nil
	}
	out := new(UUIDList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *UUIDList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UUIDSpec) DeepCopyInto(out *UUIDSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UUIDSpec.
func (in *UUIDSpec) DeepCopy() *UUIDSpec {
	if in == nil {
		return nil
	}
	out := new(UUIDSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VaultDynamicSecret) DeepCopyInto(out *VaultDynamicSecret) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VaultDynamicSecret.
func (in *VaultDynamicSecret) DeepCopy() *VaultDynamicSecret {
	if in == nil {
		return nil
	}
	out := new(VaultDynamicSecret)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VaultDynamicSecret) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VaultDynamicSecretList) DeepCopyInto(out *VaultDynamicSecretList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]VaultDynamicSecret, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VaultDynamicSecretList.
func (in *VaultDynamicSecretList) DeepCopy() *VaultDynamicSecretList {
	if in == nil {
		return nil
	}
	out := new(VaultDynamicSecretList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VaultDynamicSecretList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VaultDynamicSecretSpec) DeepCopyInto(out *VaultDynamicSecretSpec) {
	*out = *in
	if in.Parameters != nil {
		in, out := &in.Parameters, &out.Parameters
		*out = new(apiextensionsv1.JSON)
		(*in).DeepCopyInto(*out)
	}
	if in.Provider != nil {
		in, out := &in.Provider, &out.Provider
		*out = new(v1beta1.VaultProvider)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VaultDynamicSecretSpec.
func (in *VaultDynamicSecretSpec) DeepCopy() *VaultDynamicSecretSpec {
	if in == nil {
		return nil
	}
	out := new(VaultDynamicSecretSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Webhook) DeepCopyInto(out *Webhook) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Webhook.
func (in *Webhook) DeepCopy() *Webhook {
	if in == nil {
		return nil
	}
	out := new(Webhook)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Webhook) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WebhookCAProvider) DeepCopyInto(out *WebhookCAProvider) {
	*out = *in
	if in.Namespace != nil {
		in, out := &in.Namespace, &out.Namespace
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WebhookCAProvider.
func (in *WebhookCAProvider) DeepCopy() *WebhookCAProvider {
	if in == nil {
		return nil
	}
	out := new(WebhookCAProvider)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WebhookList) DeepCopyInto(out *WebhookList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Webhook, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WebhookList.
func (in *WebhookList) DeepCopy() *WebhookList {
	if in == nil {
		return nil
	}
	out := new(WebhookList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *WebhookList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WebhookResult) DeepCopyInto(out *WebhookResult) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WebhookResult.
func (in *WebhookResult) DeepCopy() *WebhookResult {
	if in == nil {
		return nil
	}
	out := new(WebhookResult)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WebhookSecret) DeepCopyInto(out *WebhookSecret) {
	*out = *in
	out.SecretRef = in.SecretRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WebhookSecret.
func (in *WebhookSecret) DeepCopy() *WebhookSecret {
	if in == nil {
		return nil
	}
	out := new(WebhookSecret)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WebhookSpec) DeepCopyInto(out *WebhookSpec) {
	*out = *in
	if in.Headers != nil {
		in, out := &in.Headers, &out.Headers
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Timeout != nil {
		in, out := &in.Timeout, &out.Timeout
		*out = new(metav1.Duration)
		**out = **in
	}
	out.Result = in.Result
	if in.Secrets != nil {
		in, out := &in.Secrets, &out.Secrets
		*out = make([]WebhookSecret, len(*in))
		copy(*out, *in)
	}
	if in.CABundle != nil {
		in, out := &in.CABundle, &out.CABundle
		*out = make([]byte, len(*in))
		copy(*out, *in)
	}
	if in.CAProvider != nil {
		in, out := &in.CAProvider, &out.CAProvider
		*out = new(WebhookCAProvider)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WebhookSpec.
func (in *WebhookSpec) DeepCopy() *WebhookSpec {
	if in == nil {
		return nil
	}
	out := new(WebhookSpec)
	in.DeepCopyInto(out)
	return out
}
