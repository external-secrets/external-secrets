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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// ClusterProviderClassSpec defines the desired state of ClusterProviderClass.
type ClusterProviderClassSpec struct {
	// +kubebuilder:validation:MinLength:=1
	Address string `json:"address"`
}

// ClusterProviderClassStatus defines the observed state of ClusterProviderClass.
type ClusterProviderClassStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={externalsecrets},shortName=cpc
// +kubebuilder:printcolumn:name="Address",type=string,JSONPath=`.spec.address`

// ClusterProviderClass is a cluster-scoped store runtime class.
type ClusterProviderClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterProviderClassSpec   `json:"spec"`
	Status ClusterProviderClassStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterProviderClassList contains a list of ClusterProviderClass.
type ClusterProviderClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterProviderClass `json:"items"`
}
