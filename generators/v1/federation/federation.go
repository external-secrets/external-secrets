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

// Package federation implements federation generator.
package federation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

const (
	errNoSpec          = "no config spec provided"
	errParseSpec       = "unable to parse spec: %w"
	errFetchSecretRef  = "could not fetch secret ref: %w"
	errFederationCall  = "failed to call federation server: %w"
	errInvalidResponse = "invalid response from federation server: %w"
	errCreateRequest   = "failed to create request: %w"
)

// Generator implements the generator interface for federation.
type Generator struct{}

// Generate implements the Generator interface.
func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}

	spec, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}

	// Get auth token from k8s secret
	authToken, err := getFromSecretRef(ctx, spec.Spec.Auth.TokenSecretRef, resolvers.EmptyStoreKind, kube, namespace)
	if err != nil {
		return nil, nil, fmt.Errorf(errFetchSecretRef, err)
	}

	var caCert string
	if spec.Spec.Auth.CACertSecretRef != nil {
		caCert, err = getFromSecretRef(ctx, spec.Spec.Auth.CACertSecretRef, resolvers.EmptyStoreKind, kube, namespace)
		if err != nil {
			return nil, nil, fmt.Errorf(errFetchSecretRef, err)
		}
	}

	url := buildFederationURL(spec)

	payload := map[string]string{}
	if caCert != "" {
		payload["ca.crt"] = caCert
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Make the HTTP request
	resp, err := makeHTTPRequest(ctx, "POST", url, authToken, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, nil, err
	}
	defer closeResponseBody(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, nil, handleErrorResponse(resp)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf(errInvalidResponse, err)
	}

	byteMap := make(map[string][]byte)
	for k, v := range result {
		byteMap[k] = []byte(v)
	}

	return byteMap, nil, nil
}

// Cleanup implements the Generator interface.
func (g *Generator) Cleanup(ctx context.Context, jsonSpec *apiextensions.JSON, _ genv1alpha1.GeneratorProviderState, kclient client.Client, namespace string) error {
	if jsonSpec == nil {
		return errors.New(errNoSpec)
	}

	spec, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return fmt.Errorf(errParseSpec, err)
	}

	authToken, err := getFromSecretRef(ctx, spec.Spec.Auth.TokenSecretRef, resolvers.EmptyStoreKind, kclient, namespace)
	if err != nil {
		return fmt.Errorf(errFetchSecretRef, err)
	}

	url := buildFederationURL(spec)

	resp, err := makeHTTPRequest(ctx, "DELETE", url, authToken, nil)
	if err != nil {
		return err
	}
	defer closeResponseBody(resp)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return handleErrorResponse(resp)
	}

	return nil
}

// GetCleanupPolicy returns the cleanup policy for this generator.
func (g *Generator) GetCleanupPolicy(_ *apiextensions.JSON) (*genv1alpha1.CleanupPolicy, error) {
	return nil, nil
}

// LastActivityTime returns the last activity time for generated resources.
func (g *Generator) LastActivityTime(_ context.Context, _ *apiextensions.JSON, _ genv1alpha1.GeneratorProviderState, _ client.Client, _ string) (time.Time, bool, error) {
	return time.Time{}, false, nil
}

// GetKeys returns the keys generated by this generator.
func (g *Generator) GetKeys() map[string]string {
	return map[string]string{
		"<key>": "Key returned dynamically by the federated generator endpoint",
	}
}

// Helper functions.
func parseSpec(data []byte) (*genv1alpha1.Federation, error) {
	var spec genv1alpha1.Federation
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func getFromSecretRef(ctx context.Context, keySelector *esmeta.SecretKeySelector, storeKind string, kube client.Client, namespace string) (string, error) {
	if keySelector == nil {
		return "", errors.New("secret reference is nil")
	}

	value, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, keySelector)
	if err != nil {
		return "", fmt.Errorf(errFetchSecretRef, err)
	}

	return value, err
}

// buildFederationURL constructs the URL for the federation server endpoint.
func buildFederationURL(spec *genv1alpha1.Federation) string {
	return fmt.Sprintf("%s/generators/%s/%s/%s",
		spec.Spec.Server.URL,
		spec.Spec.Generator.Namespace,
		spec.Spec.Generator.Kind,
		spec.Spec.Generator.Name)
}

// makeHTTPRequest creates and sends an HTTP request to the federation server.
func makeHTTPRequest(ctx context.Context, method, url, authToken string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf(errCreateRequest, err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf(errFederationCall, err)
	}

	return resp, nil
}

// closeResponseBody safely closes the response body.
func closeResponseBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Printf("Error closing response body: %v\n", closeErr)
		}
	}
}

// handleErrorResponse reads the response body and creates an appropriate error message.
func handleErrorResponse(resp *http.Response) error {
	bodyBytes, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("federation server returned non-OK status: %d, body: %s", resp.StatusCode, string(bodyBytes))
}

// NewGenerator creates a new Generator instance.
func NewGenerator() genv1alpha1.Generator {
	return &Generator{}
}

// Kind returns the generator kind.
func Kind() string {
	return genv1alpha1.FederationKind
}
