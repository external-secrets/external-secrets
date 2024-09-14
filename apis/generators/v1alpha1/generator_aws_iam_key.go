//Copyright External Secrets Inc. All Rights Reserved

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IAMKeysSpec struct {
	// Region specifies the region to operate in.
	Region string `json:"region"`

	// Auth defines how to authenticate with AWS
	// +optional
	Auth AWSAuth `json:"auth,omitempty"`

	// You can assume a role before making calls to the
	// desired AWS service.
	// +optional
	Role string `json:"role,omitempty"`

	IAMRef IAMRef `json:"iamRef"`
}

type IAMRef struct {
	Username string `json:"username"`
	MaxKeys  int    `json:"maxKeys"`
}

// IAMKeys uses the CreateAccessKey API to retrieve an
// access key. It also rotates the key by making sure only X keys exist on a given user.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={awsiamkey},shortName=awsiamkey
type AWSIAMKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec IAMKeysSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// IAMKeysList contains a list of IAMKeys resources.
type AWSIAMKeysList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AWSIAMKey `json:"items"`
}
