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
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// FakeSpec contains the static data.
type FakeSpec struct {
	// Used to select the correct ESO controller (think: ingress.ingressClassName)
	// The ESO controller is instantiated with a specific controller name and filters ES based on this property
	// +optional
	Controller string `json:"controller,omitempty"`

	// Used to configure http retries if failed
	// +optional
	RetrySettings *esmeta.RetrySettings `json:"retrySettings,omitempty"`

	// Used to configure store refresh interval in seconds. Empty or 0 will default to the controller config.
	// +optional
	RefreshInterval int                `json:"refreshInterval,omitempty"`
	Data            []FakeProviderData `json:"data"`
}

type FakeProviderData struct {
	Key      string            `json:"key"`
	Value    string            `json:"value,omitempty"`
	ValueMap map[string]string `json:"valueMap,omitempty"`
	Version  string            `json:"version,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={fake},shortName=fake
type Fake struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec FakeSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// FakeList contains a list of ExternalSecret resources.
type FakeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Fake `json:"items"`
}

func init() {
}

// Fake type metadata.
var (
	FakeKind             = reflect.TypeOf(Fake{}).Name()
	FakeGroupKind        = schema.GroupKind{Group: Group, Kind: FakeKind}.String()
	FakeKindAPIVersion   = FakeKind + "." + SchemeGroupVersion.String()
	FakeGroupVersionKind = SchemeGroupVersion.WithKind(FakeKind)
)
