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
// custom resources. Use ServiceAccountName for legacy in-cluster auth, or
// server/auth/authRef for the same connection options as the Kubernetes provider.
type CRDProvider struct {
	// ServiceAccountName is the name of the ServiceAccount used in legacy mode
	// (no server URL, auth, or authRef): the controller uses in-cluster config
	// and mints a token for this account. Ignored when server, auth, or authRef
	// is set; use auth.serviceAccount instead for explicit connection mode.
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

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

	// RemoteNamespace is the namespace used for namespaced API calls (Get/List).
	// When empty, the SecretStore namespace (or cluster scope for ClusterSecretStore)
	// is used, matching previous behavior.
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
