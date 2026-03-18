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
	// Name is an optional regular expression matched against the requested
	// resource object name.
	// +optional
	Name string `json:"name,omitempty"`

	// Properties is an optional list of regular expressions matched against
	// requested property keys (for example: "spec.secretValue").
	// +optional
	Properties []string `json:"properties,omitempty"`
}

// CRDProviderWhitelist configures allow-list rules for CRD reads.
type CRDProviderWhitelist struct {
	// Rules is a list of allow rules. If rules are set, at least one rule must
	// match for a request to be allowed.
	// +optional
	Rules []CRDProviderWhitelistRule `json:"rules,omitempty"`
}

// CRDProvider configures a store to fetch data from arbitrary Kubernetes
// custom resources in the local cluster, authenticated as a ServiceAccount.
type CRDProvider struct {
	// ServiceAccountName is the name of the ServiceAccount used when
	// accessing the Kubernetes API. The ServiceAccount must exist in the same
	// namespace as the ExternalSecret (or in the controller namespace for a
	// ClusterSecretStore).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	ServiceAccountName string `json:"serviceAccountName"`

	// Resource identifies the CRD by its API group, version and kind.
	// +kubebuilder:validation:Required
	Resource CRDProviderResource `json:"resource"`

	// Whitelist optionally restricts which object names and requested properties
	// are allowed to be read.
	// +optional
	Whitelist *CRDProviderWhitelist `json:"whitelist,omitempty"`
}
