/*
Copyright © 2025 ESO Maintainer Team

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

// Package metadata provides functionality for handling metadata for pushed secrets.
package metadata

import (
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"
)

const (
	// APIVersion is the apiVersion for PushSecretMetadata.
	APIVersion = "kubernetes.external-secrets.io/v1alpha1"
	// Kind is the kind for PushSecretMetadata.
	Kind = "PushSecretMetadata"
)

// PushSecretMetadata represents metadata associated with a pushed secret.
// T represents the type of custom metadata that can be associated with the secret.
type PushSecretMetadata[T any] struct {
	// Kind is the type of the resource.
	Kind string `json:"kind"`
	// APIVersion is the version of the API.
	APIVersion string `json:"apiVersion"`
	// Spec holds the specific metadata for the pushed secret.
	Spec T `json:"spec,omitempty"`
}

// ParseMetadataParameters parses metadata with an arbitrary Spec.
func ParseMetadataParameters[T any](data *apiextensionsv1.JSON) (*PushSecretMetadata[T], error) {
	if data == nil {
		return nil, nil
	}
	var metadata PushSecretMetadata[T]
	err := yaml.Unmarshal(data.Raw, &metadata, yaml.DisallowUnknownFields)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s %s: %w", APIVersion, Kind, err)
	}

	if metadata.APIVersion != APIVersion {
		return nil, fmt.Errorf("unexpected apiVersion %q, expected %q", metadata.APIVersion, APIVersion)
	}

	if metadata.Kind != Kind {
		return nil, fmt.Errorf("unexpected kind %q, expected %q", metadata.Kind, Kind)
	}

	return &metadata, nil
}
