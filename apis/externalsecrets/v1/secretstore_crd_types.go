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

package v1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// CRDProviderResource identifies a Kubernetes custom resource by its full
// API coordinates: group, version and kind.
type CRDProviderResource struct {
	// Group is the API group of the resource (e.g. "config.example.io").
	// Use an empty string for core Kubernetes resources such as ConfigMap.
	// +optional
	Group string `json:"group,omitempty"`

	// Version is the API version of the resource (e.g. "v1alpha1").
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Version string `json:"version"`

	// Kind is the Kubernetes resource kind (e.g. "MyCustomResource").
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Kind string `json:"kind"`
}

// CRDProviderWhitelistRule defines a single allow rule for CRD reads.
type CRDProviderWhitelistRule struct {
	// Name is an optional regular expression matched against the bare object name.
	// For both SecretStore and ClusterSecretStore this is always the object name
	// without any namespace prefix (e.g. "my-db-spec", not "prod/my-db-spec").
	// +optional
	Name string `json:"name,omitempty"`

	// Namespace is an optional regular expression matched against the namespace of
	// the object. Applies only when a ClusterSecretStore is used; it is ignored
	// for SecretStore (where the namespace is fixed to the store namespace).
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Properties is an optional list of regular expressions matched against
	// requested property keys (for example: "spec.secretValue").
	// +optional
	Properties []string `json:"properties,omitempty"`
}

// CRDProviderWhitelist configures allow-list rules for CRD reads.
// If any rules are present, a request must satisfy ALL non-empty filters of at
// least one rule; requests that match no rule are denied.
type CRDProviderWhitelist struct {
	// Rules is a list of allow rules. If rules are set, at least one rule must
	// match for a request to be allowed.
	// +optional
	Rules []CRDProviderWhitelistRule `json:"rules,omitempty"`
}

// CRDProvider configures a store to fetch data from arbitrary Kubernetes
// custom resources.
//
// # Authentication modes
//
// Legacy (in-cluster): set serviceAccountRef only. The controller uses in-cluster
// config and mints a short-lived token for the referenced ServiceAccount, which is
// then used to read CRDs from the local cluster.
//
// Explicit (remote cluster): set server + auth (or authRef). The auth.serviceAccount
// identifies the SA whose token is used to authenticate against the remote cluster.
// Optionally, set serviceAccountRef to impersonate a different identity on that remote
// cluster: the controller will set the Kubernetes Impersonate-User header to
// "system:serviceaccount/<namespace>/<name>" after connecting.
//
// # Remote reference keys
//
//   - SecretStore: the key is the object name only; '/' is not allowed. The API
//     namespace is always the store namespace (ExternalSecret namespace, or
//     remoteNamespace when set), never part of the key.
//   - ClusterSecretStore: use "namespace/objectName" to read a namespaced CR;
//     a key without '/' addresses a cluster-scoped CR by object name. For
//     dataFrom Find with a namespaced kind, listing uses all namespaces unless
//     remoteNamespace is set, and result keys are "namespace/objectName".
type CRDProvider struct {
	// ServiceAccountRef references the ServiceAccount used for authentication.
	//
	// Legacy mode (no server/auth/authRef): the controller mints a short-lived
	// token for this SA and uses it against the local cluster. For SecretStore
	// the namespace field is ignored (the SA must be in the store's namespace).
	// For ClusterSecretStore, namespace is required so the controller knows
	// where the SA lives; when omitted it defaults to "default".
	//
	// Explicit mode (server + auth or authRef): serviceAccountRef is optional.
	// When set, the controller impersonates this SA on the remote cluster after
	// connecting via auth/authRef. For SecretStore the SA namespace is the store
	// namespace; for ClusterSecretStore, namespace must be set explicitly.
	// +optional
	ServiceAccountRef *esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`

	// Server configures the Kubernetes API address and TLS trust, same as the
	// Kubernetes provider.
	// +optional
	Server KubernetesServer `json:"server,omitempty"`

	// Auth configures authentication to the Kubernetes API, same as the
	// Kubernetes provider. Required when Server.URL is set (unless using AuthRef).
	// +optional
	Auth *KubernetesAuth `json:"auth,omitempty"`

	// AuthRef references a Secret containing a kubeconfig. Same semantics as the
	// Kubernetes provider.
	// +optional
	AuthRef *esmeta.SecretKeySelector `json:"authRef,omitempty"`

	// RemoteNamespace is the default namespace for namespaced API calls on SecretStore.
	// For ClusterSecretStore with a namespaced resource, when set it limits dataFrom
	// List to that namespace (keys in the result map are object names only). When
	// empty, List spans all namespaces and keys are namespace/objectName. Per-entry
	// namespace for Get uses remoteRef.key namespace/objectName for ClusterSecretStore.
	// +optional
	// +kubebuilder:default=default
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=63
	// +kubebuilder:validation:Pattern:=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	RemoteNamespace string `json:"remoteNamespace,omitempty"`

	// Resource identifies the CRD by its API group, version and kind.
	// +kubebuilder:validation:Required
	Resource CRDProviderResource `json:"resource"`

	// Whitelist optionally restricts which object names and requested properties
	// are allowed to be read.
	// +optional
	Whitelist *CRDProviderWhitelist `json:"whitelist,omitempty"`
}
