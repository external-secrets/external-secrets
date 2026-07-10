/*
Copyright © The ESO Authors

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

// CRDProviderResource identifies a Kubernetes resource (CRD or core) by its
// full API coordinates: group, version and kind.
type CRDProviderResource struct {
	// Group is the API group of the resource. Use "" (empty string) for core
	// Kubernetes resources such as ConfigMap; use e.g. "config.example.io"
	// for a CRD. The field is required to be present in the manifest — write
	// `group: ""` explicitly for core resources so typos fail at admission
	// time rather than later at discovery.
	// +kubebuilder:validation:Required
	Group string `json:"group"`

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
// resources, including both custom resources (CRDs) and core API resources
// (e.g. ConfigMap, addressed by setting resource.group to ""). Kubernetes
// Secrets are intentionally blocked; use the Kubernetes provider for those.
//
// # Authentication modes
//
// In-cluster: set auth.serviceAccount and omit server. The server URL defaults
// to the in-cluster API (kubernetes.default) and the controller mints a
// short-lived token for the referenced ServiceAccount to read CRDs locally.
//
// Remote cluster: set server plus auth (serviceAccount, token, or cert) or
// authRef (a kubeconfig Secret), exactly like the Kubernetes provider.
//
// # Remote reference keys
//
//   - SecretStore: the key is the object name only; '/' is not allowed. The API
//     namespace is always the store namespace, never part of the key.
//   - ClusterSecretStore: use "namespace/objectName" to read a namespaced CR;
//     a key without '/' addresses a cluster-scoped CR by object name. For
//     dataFrom Find with a namespaced kind, listing spans all namespaces and
//     result keys are "namespace/objectName".
//
// +kubebuilder:validation:AtMostOneOf=auth;authRef
// +kubebuilder:validation:XValidation:rule="has(self.auth) || has(self.authRef)",message="one of auth or authRef is required"
type CRDProvider struct {
	// Server configures the Kubernetes API address and TLS trust, same as the
	// Kubernetes provider. When omitted, the URL defaults to the in-cluster API.
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

	// Resource identifies the CRD by its API group, version and kind.
	// +kubebuilder:validation:Required
	Resource CRDProviderResource `json:"resource"`

	// Whitelist optionally restricts which object names and requested properties
	// are allowed to be read.
	// +optional
	Whitelist *CRDProviderWhitelist `json:"whitelist,omitempty"`
}
