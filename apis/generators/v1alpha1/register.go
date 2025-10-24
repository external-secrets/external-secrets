/*
Copyright © 2025 ESO Maintainer Team

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

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

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

var (
	// ECRAuthorizationTokenKind is the kind name for ECRAuthorizationToken resource.
	ECRAuthorizationTokenKind = reflect.TypeOf(ECRAuthorizationToken{}).Name()
	// STSSessionTokenKind is the kind name for STSSessionToken resource.
	STSSessionTokenKind = reflect.TypeOf(STSSessionToken{}).Name()
	// GCRAccessTokenKind is the kind name for GCRAccessToken resource.
	GCRAccessTokenKind = reflect.TypeOf(GCRAccessToken{}).Name()
	// ACRAccessTokenKind is the kind name for ACRAccessToken resource.
	ACRAccessTokenKind = reflect.TypeOf(ACRAccessToken{}).Name()
	// PasswordKind is the kind name for Password resource.
	PasswordKind = reflect.TypeOf(Password{}).Name()
	// SSHKeyKind is the kind name for SSHKey resource.
	SSHKeyKind = reflect.TypeOf(SSHKey{}).Name()
	// WebhookKind is the kind name for Webhook resource.
	WebhookKind = reflect.TypeOf(Webhook{}).Name()
	// FakeKind is the kind name for Fake resource.
	FakeKind = reflect.TypeOf(Fake{}).Name()
	// VaultDynamicSecretKind is the kind name for VaultDynamicSecret resource.
	VaultDynamicSecretKind = reflect.TypeOf(VaultDynamicSecret{}).Name()
	// GithubAccessTokenKind is the kind name for GithubAccessToken resource.
	GithubAccessTokenKind = reflect.TypeOf(GithubAccessToken{}).Name()
	// QuayAccessTokenKind is the kind name for QuayAccessToken resource.
	QuayAccessTokenKind = reflect.TypeOf(QuayAccessToken{}).Name()
	// UUIDKind is the kind name for UUID resource.
	UUIDKind = reflect.TypeOf(UUID{}).Name()
	// GrafanaKind is the kind name for Grafana resource.
	GrafanaKind = reflect.TypeOf(Grafana{}).Name()
	// MFAKind is the kind name for MFA resource.
	MFAKind = reflect.TypeOf(MFA{}).Name()
	// ClusterGeneratorKind is the kind name for ClusterGenerator resource.
	ClusterGeneratorKind = reflect.TypeOf(ClusterGenerator{}).Name()
	// CloudsmithAccessTokenKind is the kind name for CloudsmithAccessToken resource.
	CloudsmithAccessTokenKind = reflect.TypeOf(CloudsmithAccessToken{}).Name()
)

func init() {
	SchemeBuilder.Register(&GeneratorState{}, &GeneratorStateList{})

	/*
		===============================================================================
		 NOTE: when adding support for new kinds of generators:
		  1. register the struct types in `SchemeBuilder` (right below this note)
		  2. update the `kubebuilder:validation:Enum` annotation for GeneratorRef.Kind (apis/externalsecrets/v1beta1/externalsecret_types.go)
		  3. add it to the imports of (pkg/generator/register/register.go)
		  4. add it to the ClusterRole called "*-controller" (deploy/charts/external-secrets/templates/rbac.yaml)
		  5. support it in ClusterGenerator:
			  - add a new GeneratorKind enum value (apis/generators/v1alpha1/types_cluster.go)
			  - update the `kubebuilder:validation:Enum` annotation for the GeneratorKind enum
			  - add a spec field to GeneratorSpec (apis/generators/v1alpha1/types_cluster.go)
			  - update the clusterGeneratorToVirtual() function (pkg/utils/resolvers/generator.go)
		===============================================================================
	*/

	SchemeBuilder.Register(&ACRAccessToken{}, &ACRAccessTokenList{})
	SchemeBuilder.Register(&ClusterGenerator{}, &ClusterGeneratorList{})
	SchemeBuilder.Register(&CloudsmithAccessToken{}, &CloudsmithAccessTokenList{})
	SchemeBuilder.Register(&ECRAuthorizationToken{}, &ECRAuthorizationTokenList{})
	SchemeBuilder.Register(&Fake{}, &FakeList{})
	SchemeBuilder.Register(&GCRAccessToken{}, &GCRAccessTokenList{})
	SchemeBuilder.Register(&GithubAccessToken{}, &GithubAccessTokenList{})
	SchemeBuilder.Register(&QuayAccessToken{}, &QuayAccessTokenList{})
	SchemeBuilder.Register(&Password{}, &PasswordList{})
	SchemeBuilder.Register(&SSHKey{}, &SSHKeyList{})
	SchemeBuilder.Register(&STSSessionToken{}, &STSSessionTokenList{})
	SchemeBuilder.Register(&UUID{}, &UUIDList{})
	SchemeBuilder.Register(&VaultDynamicSecret{}, &VaultDynamicSecretList{})
	SchemeBuilder.Register(&Webhook{}, &WebhookList{})
	SchemeBuilder.Register(&Grafana{}, &GrafanaList{})
	SchemeBuilder.Register(&MFA{}, &MFAList{})
}
