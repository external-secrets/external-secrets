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

package metadata

import (
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"
)

const (
	APIVersion = "kubernetes.external-secrets.io/v1alpha1"
	Kind       = "PushSecretMetadata"
)

type PushSecretMetadata[T any] struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Spec       T      `json:"spec,omitempty"`
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
