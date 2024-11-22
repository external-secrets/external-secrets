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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type ControllerClassResource struct {
	Spec struct {
		ControllerClass string `json:"controller"`
	} `json:"spec"`
}

type ClusterGeneratorSpec struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	ACRAccessToken *ACRAccessToken `json:"acrAccessToken,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	ECRAuthorizationToken *ECRAuthorizationToken `json:"ecrRAuthorizationToken,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Fake *Fake `json:"fake,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	GCRAccessToken *GCRAccessToken `json:"gcrAccessToken,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	GithubAccessToken *GithubAccessToken `json:"githubAccessToken,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Password *Password `json:"password,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	STSSessionToken *STSSessionToken `json:"stsSessionToken,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	UUID *UUID `json:"uuid,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	VaultDynamicSecret *VaultDynamicSecret `json:"vaultDynamicSecret,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Webhook *Webhook `json:"webhook,omitempty"`
}

type ClusterGeneratorStatus struct{}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// ClusterGenerator represents a cluster-wide generator which can be referenced as part of `generatorRef` fields.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Cluster,categories={external-secrets, external-secrets-generators},shortName=cg
type ClusterGenerator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterGeneratorSpec   `json:"spec,omitempty"`
	Status ClusterGeneratorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterGeneratorList contains a list of ClusterGenerator resources.
type ClusterGeneratorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterGenerator `json:"items"`
}
