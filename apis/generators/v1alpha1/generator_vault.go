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
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

type VaultDynamicSecretSpec struct {
	// Used to select the correct ESO controller (think: ingress.ingressClassName)
	// The ESO controller is instantiated with a specific controller name and filters VDS based on this property
	// +optional
	Controller string `json:"controller,omitempty"`

	// Vault API method to use (GET/POST/other)
	Method string `json:"method,omitempty"`

	// Parameters to pass to Vault write (for non-GET methods)
	Parameters *apiextensions.JSON `json:"parameters,omitempty"`

	// Result type defines which data is returned from the generator.
	// By default it is the "data" section of the Vault API response.
	// When using e.g. /auth/token/create the "data" section is empty but
	// the "auth" section contains the generated token.
	// Please refer to the vault docs regarding the result data structure.
	// +kubebuilder:default=Data
	ResultType VaultDynamicSecretResultType `json:"resultType,omitempty"`

	// Vault provider common spec
	Provider *esv1beta1.VaultProvider `json:"provider"`

	// Vault path to obtain the dynamic secret from
	Path string `json:"path"`
}

// +kubebuilder:validation:Enum=Data;Auth
type VaultDynamicSecretResultType string

const (
	VaultDynamicSecretResultTypeData VaultDynamicSecretResultType = "Data"
	VaultDynamicSecretResultTypeAuth VaultDynamicSecretResultType = "Auth"
)

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={vaultdynamicsecret},shortName=vaultdynamicsecret
type VaultDynamicSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VaultDynamicSecretSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type VaultDynamicSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VaultDynamicSecret `json:"items"`
}
