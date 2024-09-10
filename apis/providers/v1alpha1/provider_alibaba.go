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

// AlibabaAuth contains a secretRef for credentials.
type AlibabaAuth struct {
	// +optional
	SecretRef *AlibabaAuthSecretRef `json:"secretRef,omitempty"`
	// +optional
	RRSAAuth *AlibabaRRSAAuth `json:"rrsa,omitempty"`
}

// AlibabaAuthSecretRef holds secret references for Alibaba credentials.
type AlibabaAuthSecretRef struct {
	// The AccessKeyID is used for authentication
	AccessKeyID esmeta.SecretKeySelector `json:"accessKeyIDSecretRef"`
	// The AccessKeySecret is used for authentication
	AccessKeySecret esmeta.SecretKeySelector `json:"accessKeySecretSecretRef"`
}

// Authenticate against Alibaba using RRSA.
type AlibabaRRSAAuth struct {
	OIDCProviderARN   string `json:"oidcProviderArn"`
	OIDCTokenFilePath string `json:"oidcTokenFilePath"`
	RoleARN           string `json:"roleArn"`
	SessionName       string `json:"sessionName"`
}

// AlibabaProvider configures a store to sync secrets using the Alibaba Secret Manager provider.
type AlibabaSpec struct {
	// Used to select the correct ESO controller (think: ingress.ingressClassName)
	// The ESO controller is instantiated with a specific controller name and filters ES based on this property
	// +optional
	Controller string `json:"controller,omitempty"`

	// Used to configure http retries if failed
	// +optional
	RetrySettings *esmeta.RetrySettings `json:"retrySettings,omitempty"`

	// Used to configure store refresh interval in seconds. Empty or 0 will default to the controller config.
	// +optional
	RefreshInterval int `json:"refreshInterval,omitempty"`

	Auth AlibabaAuth `json:"auth"`
	// Alibaba Region to be used for the provider
	RegionID string `json:"regionID"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={alibaba},shortName=alibaba
type Alibaba struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AlibabaSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// AlibabaList contains a list of Alibaba resources.
type AlibabaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Alibaba `json:"items"`
}

func init() {
}

// Alibaba type metadata.
var (
	AlibabaKind             = reflect.TypeOf(Alibaba{}).Name()
	AlibabaGroupKind        = schema.GroupKind{Group: Group, Kind: AlibabaKind}.String()
	AlibabaKindAPIVersion   = AlibabaKind + "." + SchemeGroupVersion.String()
	AlibabaGroupVersionKind = SchemeGroupVersion.WithKind(AlibabaKind)
)
