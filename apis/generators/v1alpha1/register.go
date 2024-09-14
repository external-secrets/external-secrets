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

package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	Group   = "generators.external-secrets.io"
	Version = "v1alpha1"
)

var (
	// SchemeGroupVersion is group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme
)

// ECRAuthorizationToken type metadata.
var (
	ECRAuthorizationTokenKind             = reflect.TypeOf(ECRAuthorizationToken{}).Name()
	ECRAuthorizationTokenGroupKind        = schema.GroupKind{Group: Group, Kind: ECRAuthorizationTokenKind}.String()
	ECRAuthorizationTokenKindAPIVersion   = ECRAuthorizationTokenKind + "." + SchemeGroupVersion.String()
	ECRAuthorizationTokenGroupVersionKind = SchemeGroupVersion.WithKind(ECRAuthorizationTokenKind)
)

// GCRAccessToken type metadata.
var (
	GCRAccessTokenKind             = reflect.TypeOf(GCRAccessToken{}).Name()
	GCRAccessTokenGroupKind        = schema.GroupKind{Group: Group, Kind: GCRAccessTokenKind}.String()
	GCRAccessTokenKindAPIVersion   = GCRAccessTokenKind + "." + SchemeGroupVersion.String()
	GCRAccessTokenGroupVersionKind = SchemeGroupVersion.WithKind(GCRAccessTokenKind)
)

// ACRAccessToken type metadata.
var (
	ACRAccessTokenKind             = reflect.TypeOf(ACRAccessToken{}).Name()
	ACRAccessTokenGroupKind        = schema.GroupKind{Group: Group, Kind: ACRAccessTokenKind}.String()
	ACRAccessTokenKindAPIVersion   = ACRAccessTokenKind + "." + SchemeGroupVersion.String()
	ACRAccessTokenGroupVersionKind = SchemeGroupVersion.WithKind(ACRAccessTokenKind)
)

// Password type metadata.
var (
	PasswordKind             = reflect.TypeOf(Password{}).Name()
	PasswordGroupKind        = schema.GroupKind{Group: Group, Kind: PasswordKind}.String()
	PasswordKindAPIVersion   = PasswordKind + "." + SchemeGroupVersion.String()
	PasswordGroupVersionKind = SchemeGroupVersion.WithKind(PasswordKind)
)

// Webhook type metadata.
var (
	WebhookKind             = reflect.TypeOf(Webhook{}).Name()
	WebhookGroupKind        = schema.GroupKind{Group: Group, Kind: WebhookKind}.String()
	WebhookKindAPIVersion   = WebhookKind + "." + SchemeGroupVersion.String()
	WebhookGroupVersionKind = SchemeGroupVersion.WithKind(WebhookKind)
)

// Fake type metadata.
var (
	FakeKind             = reflect.TypeOf(Fake{}).Name()
	FakeGroupKind        = schema.GroupKind{Group: Group, Kind: FakeKind}.String()
	FakeKindAPIVersion   = FakeKind + "." + SchemeGroupVersion.String()
	FakeGroupVersionKind = SchemeGroupVersion.WithKind(FakeKind)
)

// Vault type metadata.
var (
	VaultDynamicSecretKind             = reflect.TypeOf(VaultDynamicSecret{}).Name()
	VaultDynamicSecretGroupKind        = schema.GroupKind{Group: Group, Kind: VaultDynamicSecretKind}.String()
	VaultDynamicSecretKindAPIVersion   = VaultDynamicSecretKind + "." + SchemeGroupVersion.String()
	VaultDynamicSecretGroupVersionKind = SchemeGroupVersion.WithKind(VaultDynamicSecretKind)
)

// GithubAccessToken type metadata.
var (
	GithubAccessTokenKind             = reflect.TypeOf(GithubAccessToken{}).Name()
	GithubAccessTokenGroupKind        = schema.GroupKind{Group: Group, Kind: GithubAccessTokenKind}.String()
	GithubAccessTokenKindAPIVersion   = GithubAccessTokenKind + "." + SchemeGroupVersion.String()
	GithubAccessTokenGroupVersionKind = SchemeGroupVersion.WithKind(GithubAccessTokenKind)
)

func init() {
	SchemeBuilder.Register(&ECRAuthorizationToken{}, &ECRAuthorizationToken{})
	SchemeBuilder.Register(&GCRAccessToken{}, &GCRAccessTokenList{})
	SchemeBuilder.Register(&GithubAccessToken{}, &GithubAccessTokenList{})
	SchemeBuilder.Register(&ACRAccessToken{}, &ACRAccessTokenList{})
	SchemeBuilder.Register(&Fake{}, &FakeList{})
	SchemeBuilder.Register(&VaultDynamicSecret{}, &VaultDynamicSecretList{})
	SchemeBuilder.Register(&Password{}, &PasswordList{})
	SchemeBuilder.Register(&Webhook{}, &WebhookList{})
}
