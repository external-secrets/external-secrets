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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// ServiceAccountTokenSpec controls how a Kubernetes ServiceAccount token is issued.
type ServiceAccountTokenSpec struct {
	// ServiceAccountRef names the ServiceAccount to issue a token for and carries
	// the audiences to request.
	//
	// The token is always issued in the namespace this generator is evaluated in.
	// Setting `namespace` is rejected rather than ignored, because issuing a token
	// in another namespace would let the caller reach beyond it.
	// +required
	ServiceAccountRef esmeta.ServiceAccountSelector `json:"serviceAccountRef"`

	// ExpirationSeconds is the requested lifetime of the token, in seconds.
	//
	// The issuer may return a shorter validity than requested: the API server
	// enforces its own bounds and `--service-account-max-token-expiration` caps
	// requests without reporting it. The generated `expirationTimestamp` is
	// therefore authoritative, not this field. See [TokenRequest].
	//
	// Defaults to the API server's own default when unset.
	// +optional
	//
	// [TokenRequest]: https://kubernetes.io/docs/reference/kubernetes-api/authentication-resources/token-request-v1/
	ExpirationSeconds *int64 `json:"expirationSeconds,omitempty"`
}

// ServiceAccountToken issues short-lived Kubernetes ServiceAccount tokens through the TokenRequest API.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type ServiceAccountToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ServiceAccountTokenSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceAccountTokenList contains a list of ServiceAccountToken resources.
type ServiceAccountTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceAccountToken `json:"items"`
}
