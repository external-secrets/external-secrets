package v1alpha1

import (
	"context"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:object:root=false
// +kubebuilder:object:generate:false
// +k8s:deepcopy-gen:interfaces=nil
// +k8s:deepcopy-gen=nil
type Generator interface {
	Generate(
		ctx context.Context,
		obj *apiextensions.JSON,
		kube client.Client,
		namespace string,
	) (map[string][]byte, error)
}
