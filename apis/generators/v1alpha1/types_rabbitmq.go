// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

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

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// RabbitMQSpec user generation behavior for rabbitMQ.
type RabbitMQSpec struct {
	// Server defines the RabbitMQ Server Parameters to connect to.
	//+required
	Server RabbitMQServer `json:"server"`
	// Auth defines the RabbitMQ authentication parameters.
	//+required
	Auth RabbitMQAuth `json:"auth"`
	// Config defines how to rotate the Secret within RabbitMQ.
	//+required
	Config RabbitMQConfig `json:"config"`
}

// RabbitMQServer defines the RabbitMQ Server Parameters to connect to.
type RabbitMQServer struct {
	// Host is the hostname of the RabbitMQ server.
	//+required
	Host string `json:"host"`
	// Port is the port of the RabbitMQ server.
	//+kubebuilder:default=15672
	//+optional
	Port int `json:"port,omitempty"`
	// TLS indicates whether to use TLS.
	//+kubebuilder:default=false
	//+optional
	TLS bool `json:"tls"`
}

// RabbitMQAuth defines the RabbitMQ authentication parameters.
//
//kubebuilder:validation:MinProperties=1
//kubebuilder:validation:MaxProperties=1
type RabbitMQAuth struct {
	// BasicAuth contains basic authentication credentials.
	//+optional
	BasicAuth *RabbitMQBasicAuth `json:"basicAuth,omitempty"`
}

// RabbitMQBasicAuth contains basic authentication credentials.
type RabbitMQBasicAuth struct {
	// Username is the RabbitMQ username to connect to.
	// Must have sufficient permissions for administration tasks.
	// +required
	Username string `json:"username"`
	// PasswordSecretRef is a reference to a secret containing the password.
	//+required
	PasswordSecretRef esmeta.SecretKeySelector `json:"passwordSecretRef"`
}

// RabbitMQConfig contains the configuration for password rotation.
type RabbitMQConfig struct {
	// Username contains the target username to rotate passwords for.
	//+required
	Username string `json:"username"`
	// PasswordPolicy contains the password policy to apply.
	//+required
	PasswordPolicy RabbitMQPasswordPolicy `json:"passwordPolicy"`
}

// RabbitMQPasswordPolicy contains the password policy to apply.
//
//kubebuilder:validation:MinProperties=1
//kubebuilder:validation:MaxProperties=1
type RabbitMQPasswordPolicy struct {
	// PasswordGeneratorRef is a reference to a password generator.
	//+optional
	PasswordGeneratorRef *RabbitMQPasswordGeneratorRef `json:"passwordGeneratorRef"`
	// SecretRef is a reference to a Secret Key containing the Password.
	//+optional
	SecretRef *esmeta.SecretKeySelector `json:"secretRef"`
}

// RabbitMQPasswordGeneratorRef is a reference to a password generator.
type RabbitMQPasswordGeneratorRef struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// RabbitMQ generates a random password based on the.
// configuration parameters in spec.
// You can specify the length, characterset and other attributes.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type RabbitMQ struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RabbitMQSpec    `json:"spec,omitempty"`
	Status GeneratorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RabbitMQList contains a list of RabbitMQ resources.
type RabbitMQList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RabbitMQ `json:"items"`
}
