//Copyright External Secrets Inc. All Rights Reserved

package v1beta1

import apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

// +kubebuilder:object:root=false
// +kubebuilder:object:generate:false
// +k8s:deepcopy-gen:interfaces=nil
// +k8s:deepcopy-gen=nil

// PushSecretData is an interface to allow using v1alpha1.PushSecretData content in Provider registered in v1beta1.
type PushSecretData interface {
	GetMetadata() *apiextensionsv1.JSON
	GetSecretKey() string
	GetRemoteKey() string
	GetProperty() string
}

// +kubebuilder:object:root=false
// +kubebuilder:object:generate:false
// +k8s:deepcopy-gen:interfaces=nil
// +k8s:deepcopy-gen=nil

// PushSecretRemoteRef is an interface to allow using v1alpha1.PushSecretRemoteRef in Provider registered in v1beta1.
type PushSecretRemoteRef interface {
	GetRemoteKey() string
	GetProperty() string
}
