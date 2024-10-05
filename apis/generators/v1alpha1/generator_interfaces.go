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
	"context"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:object:root=false
// +kubebuilder:object:generate:false
// +k8s:deepcopy-gen:interfaces=nil
// +k8s:deepcopy-gen=nil

// Generator is the common interface for all generators that is actually used to generate whatever is needed.
type Generator interface {
	// Generate creates a new secret or set of secrets.
	// The returned map is a mapping of secret names to their respective values.
	// The status is an optional field that can be used to store any generator-specific
	// state which can be used during the Cleanup phase.
	Generate(
		ctx context.Context,
		obj *apiextensions.JSON,
		kube client.Client,
		namespace string,
	) (map[string][]byte, GeneratorProviderState, error)

	// Cleanup deletes any resources created during the Generate phase.
	// Cleanup is idempotent and should not return an error if the resources
	// have already been deleted.
	Cleanup(
		ctx context.Context,
		obj *apiextensions.JSON,
		status GeneratorProviderState,
		kube client.Client,
		namespace string,
	) error
}

type GeneratorProviderState *apiextensions.JSON
