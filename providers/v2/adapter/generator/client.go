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

// Package generator adapts v1 generator implementations to the v2 gRPC GeneratorProvider interface.
package generator

import (
	"context"
	"encoding/json"
	"fmt"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	genpb "github.com/external-secrets/external-secrets/proto/generator"
)

// Client wraps a v2 gRPC generator client and implements the v1 Generator interface.
// This allows v2 generators to be used with the existing generator infrastructure.
type Client struct {
	client          genpb.GeneratorProviderClient
	generatorRef    *genpb.GeneratorRef
	sourceNamespace string
}

// Ensure Client implements Generator interface.
var _ genv1alpha1.Generator = &Client{}

// NewClient creates a new wrapper that adapts a v2 gRPC generator to v1 Generator interface.
func NewClient(client genpb.GeneratorProviderClient, generatorRef *genpb.GeneratorRef, sourceNamespace string) genv1alpha1.Generator {
	return &Client{
		client:          client,
		generatorRef:    generatorRef,
		sourceNamespace: sourceNamespace,
	}
}

// Generate creates a new secret or set of secrets using the v2 gRPC generator.
// The jsonSpec parameter is ignored since the generator reference is already stored.
func (w *Client) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, _ client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	// If jsonSpec is provided, we need to extract the generator reference from it
	// However, for v2 generators, we already have the reference, so we can use that
	// But we should validate that the jsonSpec matches our stored reference if provided
	if jsonSpec != nil {
		// Parse the jsonSpec to extract metadata if needed
		var meta struct {
			Metadata struct {
				Name      string `json:"name"`
				Namespace string `json:"namespace"`
			} `json:"metadata"`
			APIVersion string `json:"apiVersion"`
			Kind       string `json:"kind"`
		}
		if err := json.Unmarshal(jsonSpec.Raw, &meta); err == nil {
			// Update the generator ref with the actual resource info
			if meta.Metadata.Name != "" {
				w.generatorRef.Name = meta.Metadata.Name
			}
			if meta.Metadata.Namespace != "" {
				w.generatorRef.Namespace = meta.Metadata.Namespace
			}
			if meta.APIVersion != "" {
				w.generatorRef.ApiVersion = meta.APIVersion
			}
			if meta.Kind != "" {
				w.generatorRef.Kind = meta.Kind
			}
		}
	}

	// Use the provided namespace as the source namespace
	sourceNs := namespace
	if sourceNs == "" {
		sourceNs = w.sourceNamespace
	}

	// Call the v2 gRPC Generate
	resp, err := w.client.Generate(ctx, &genpb.GenerateRequest{
		GeneratorRef:    w.generatorRef,
		SourceNamespace: sourceNs,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to call v2 generator: %w", err)
	}

	// Convert state bytes to GeneratorProviderState
	var state genv1alpha1.GeneratorProviderState
	if len(resp.State) > 0 {
		state = &apiextensions.JSON{Raw: resp.State}
	}

	return resp.Secrets, state, nil
}

// Cleanup deletes any resources created during the Generate phase.
func (w *Client) Cleanup(ctx context.Context, jsonSpec *apiextensions.JSON, state genv1alpha1.GeneratorProviderState, _ client.Client, namespace string) error {
	// Extract state bytes
	var stateBytes []byte
	if state != nil {
		stateBytes = state.Raw
	}

	// Use the provided namespace as the source namespace
	sourceNs := namespace
	if sourceNs == "" {
		sourceNs = w.sourceNamespace
	}

	// Call the v2 gRPC Cleanup
	_, err := w.client.Cleanup(ctx, &genpb.CleanupRequest{
		GeneratorRef:    w.generatorRef,
		State:           stateBytes,
		SourceNamespace: sourceNs,
	})
	if err != nil {
		return fmt.Errorf("failed to call v2 generator cleanup: %w", err)
	}

	return nil
}
