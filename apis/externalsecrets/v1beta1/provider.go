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

package v1beta1

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	// Ready indicates that the client is configured correctly
	// and can be used.
	ValidationResultReady ValidationResult = iota

	// Unknown indicates that the client can be used
	// but information is missing and it can not be validated.
	ValidationResultUnknown

	// Error indicates that there is a misconfiguration.
	ValidationResultError
)

type ValidationResult uint8

func (v ValidationResult) String() string {
	return [...]string{"Ready", "Unknown", "Error"}[v]
}

// +kubebuilder:object:root=false
// +kubebuilder:object:generate:false
// +k8s:deepcopy-gen:interfaces=nil
// +k8s:deepcopy-gen=nil

// Provider is a common interface for interacting with secret backends.
type Provider interface {
	// NewClient constructs a SecretsManager Provider
	NewClient(ctx context.Context, store GenericStore, kube client.Client, namespace string) (SecretsClient, error)

	// ValidateStore checks if the provided store is valid
	// The provider may return a warning and an error.
	// The intended use of the warning to indicate a deprecation of behavior
	// or other type of message that is NOT a validation failure but should be noticed by the user.
	ValidateStore(store GenericStore) (admission.Warnings, error)

	// Capabilities returns the provider Capabilities (Read, Write, ReadWrite)
	Capabilities() SecretStoreCapabilities
}

// +kubebuilder:object:root=false
// +kubebuilder:object:generate:false
// +k8s:deepcopy-gen:interfaces=nil
// +k8s:deepcopy-gen=nil

// SecretsClient provides access to secrets.
type SecretsClient interface {
	// GetSecret returns a single secret from the provider
	// if GetSecret returns an error with type NoSecretError
	// then the secret entry will be deleted depending on the deletionPolicy.
	GetSecret(ctx context.Context, ref ExternalSecretDataRemoteRef) ([]byte, error)

	// PushSecret will write a single secret into the provider
	PushSecret(ctx context.Context, secret *corev1.Secret, data PushSecretData) error

	// DeleteSecret will delete the secret from a provider
	DeleteSecret(ctx context.Context, remoteRef PushSecretRemoteRef) error

	// SecretExists checks if a secret is already present in the provider at the given location.
	SecretExists(ctx context.Context, remoteRef PushSecretRemoteRef) (bool, error)

	// Validate checks if the client is configured correctly
	// and is able to retrieve secrets from the provider.
	// If the validation result is unknown it will be ignored.
	Validate() (ValidationResult, error)

	// GetSecretMap returns multiple k/v pairs from the provider
	GetSecretMap(ctx context.Context, ref ExternalSecretDataRemoteRef) (map[string][]byte, error)

	// GetAllSecrets returns multiple k/v pairs from the provider
	GetAllSecrets(ctx context.Context, ref ExternalSecretFind) (map[string][]byte, error)

	Close(ctx context.Context) error
}

var NoSecretErr = NoSecretError{}

// NoSecretError shall be returned when a GetSecret can not find the
// desired secret. This is used for deletionPolicy.
type NoSecretError struct{}

func (NoSecretError) Error() string {
	return "Secret does not exist"
}
