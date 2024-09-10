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

package v1

type ProviderRef struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
}

// ReferentCallOrigin determines if NewClientFromObject was called from the Provider, The SecretStore, or the ClusterSecretStore.
type ReferentCallOrigin string

const (
	ReferentCallSecretStore ReferentCallOrigin = "SecretStore"

	ReferentCallClusterSecretStore ReferentCallOrigin = "ClusterSecretStore"

	ReferentCallProvider ReferentCallOrigin = "Provider"
)

// A reference to a specific 'key' within a Secret resource,
// In some instances, `key` is a required field.
type SecretKeySelector struct {
	// The name of the Secret resource being referred to.
	Name string `json:"name,omitempty"`
	// Namespace of the resource being referred to. Ignored if referent is not cluster-scoped. cluster-scoped defaults
	// to the namespace of the referent.
	// +optional
	Namespace *string `json:"namespace,omitempty"`
	// The key of the entry in the Secret resource's `data` field to be used. Some instances of this field may be
	// defaulted, in others it may be required.
	// +optional
	Key string `json:"key,omitempty"`
}

// A reference to a ServiceAccount resource.
type ServiceAccountSelector struct {
	// The name of the ServiceAccount resource being referred to.
	Name string `json:"name"`
	// Namespace of the resource being referred to. Ignored if referent is not cluster-scoped. cluster-scoped defaults
	// to the namespace of the referent.
	// +optional
	Namespace *string `json:"namespace,omitempty"`
	// Audience specifies the `aud` claim for the service account token
	// If the service account uses a well-known annotation for e.g. IRSA or GCP Workload Identity
	// then this audiences will be appended to the list
	// +optional
	Audiences []string `json:"audiences,omitempty"`
}

type CAProviderType string

const (
	CAProviderTypeSecret    CAProviderType = "Secret"
	CAProviderTypeConfigMap CAProviderType = "ConfigMap"
)

// Used to provide custom certificate authority (CA) certificates
// for a secret store. The CAProvider points to a Secret or ConfigMap resource
// that contains a PEM-encoded certificate.
type CAProvider struct {
	// The type of provider to use such as "Secret", or "ConfigMap".
	// +kubebuilder:validation:Enum="Secret";"ConfigMap"
	Type CAProviderType `json:"type"`

	// The name of the object located at the provider type.
	Name string `json:"name"`

	// The key where the CA certificate can be found in the Secret or ConfigMap.
	// +kubebuilder:validation:Optional
	Key string `json:"key,omitempty"`

	// The namespace the Provider type is in.
	// Can only be defined when used in a ClusterSecretStore.
	// +optional
	Namespace *string `json:"namespace,omitempty"`
}

// Configures Retry Settings for providers/Secret Stores.
type RetrySettings struct {
	MaxRetries    *int32  `json:"maxRetries,omitempty"`
	RetryInterval *string `json:"retryInterval,omitempty"`
}
