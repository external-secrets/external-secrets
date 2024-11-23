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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// A couple of constants to define the generator's keys for accessing via Resource map values.
const (
	GeneratorGeneratorKey = "generator"
	GeneratorKindKey      = "kind"
	GeneratorSpecKey      = "spec"
)

type ControllerClassResource struct {
	Spec struct {
		ControllerClass string `json:"controller"`
	} `json:"spec"`
}

type GeneratorSpec struct {
	ACRAccessTokenSpec        *ACRAccessTokenSpec        `json:"acrAccessTokenSpec,omitempty"`
	ECRAuthorizationTokenSpec *ECRAuthorizationTokenSpec `json:"ecrRAuthorizationTokenSpec,omitempty"`
	FakeSpec                  *FakeSpec                  `json:"fakeSpec,omitempty"`
	GCRAccessTokenSpec        *GCRAccessTokenSpec        `json:"gcrAccessTokenSpec,omitempty"`
	GithubAccessTokenSpec     *GithubAccessTokenSpec     `json:"githubAccessTokenSpec,omitempty"`
	PasswordSpec              *PasswordSpec              `json:"passwordSpec,omitempty"`
	STSSessionTokenSpec       *STSSessionTokenSpec       `json:"stsSessionTokenSpec,omitempty"`
	UUIDSpec                  *UUIDSpec                  `json:"uuidSpec,omitempty"`
	VaultDynamicSecretSpec    *VaultDynamicSecretSpec    `json:"vaultDynamicSecretSpec,omitempty"`
	WebhookSpec               *WebhookSpec               `json:"webhookSpec,omitempty"`
}

type ClusterGeneratorSpec struct {
	Kind      string        `json:"kind"`
	Generator GeneratorSpec `json:"generator"`
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
