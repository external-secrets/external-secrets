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

// Package beyondtrustsecretsdynamic provides a generator for BeyondTrust Secrets Manager dynamic credentials.
package beyondtrustsecretsdynamic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	beyondtrustsecretsprovider "github.com/external-secrets/external-secrets/providers/v1/beyondtrustsecrets"
	"github.com/external-secrets/external-secrets/providers/v1/beyondtrustsecrets/httpclient"
	btsutil "github.com/external-secrets/external-secrets/providers/v1/beyondtrustsecrets/util"
)

// Generator implements BeyondTrustSecrets dynamic generator.
type Generator struct {
	// NewBeyondTrustSecretsClient is a factory function to create a BeyondTrustSecrets client.
	// If nil, defaults to httpclient.NewBeyondTrustSecretsClient.
	NewBeyondTrustSecretsClient func(server, token string) (btsutil.Client, error)
}

const (
	errNoSpec        = "no config spec provided"
	errParseSpec     = "unable to parse spec: %w"
	errMissingConfig = "no beyondtrustsecrets provider config in spec"
	errNoPath        = "path is required in spec"
	errGetSecret     = "unable to generate dynamic secret: %w"
)

// Generate creates the dynamic credentials by calling BeyondTrustSecrets generate endpoint.
func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	spec, err := getDynamicSecretSpec(jsonSpec)
	if err != nil {
		return nil, nil, err
	}
	provider := spec.Spec.Provider

	// parse and validate folder path and secret name early
	fullPath := spec.Spec.Provider.FolderPath
	folderPath, secretName := parsePath(fullPath)
	if secretName == "" {
		return nil, nil, fmt.Errorf("invalid path: missing secret name in %q", fullPath)
	}

	// create BeyondTrustSecrets provider and initialize a client for generator controller
	clientFactory := g.NewBeyondTrustSecretsClient
	if clientFactory == nil {
		clientFactory = httpclient.NewBeyondTrustSecretsClient
	}
	prov := beyondtrustsecretsprovider.Provider{
		NewBeyondTrustSecretsClient: clientFactory,
	}
	cl, err := prov.NewGeneratorClient(ctx, kube, provider, namespace)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create BeyondTrustSecrets client: %w", err)
	}

	// call generate
	generatedSecret, err := cl.GenerateDynamicSecret(ctx, secretName, folderPath)
	if err != nil {
		return nil, nil, fmt.Errorf(errGetSecret, err)
	}
	if generatedSecret == nil {
		return nil, nil, fmt.Errorf("generated secret is nil")
	}

	out := convertToByteMap(generatedSecret)

	// prepare provider state
	state := &struct {
		Path string `json:"path,omitempty"`
	}{Path: spec.Spec.Provider.FolderPath}

	stateJSON, _ := json.Marshal(state)
	gpState := genv1alpha1.GeneratorProviderState(&apiextensions.JSON{Raw: stateJSON})

	return out, gpState, nil
}

// Cleanup is a no-op for BeyondTrustSecrets dynamic generator.
func (g *Generator) Cleanup(_ context.Context, _ *apiextensions.JSON, _ genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

// getDynamicSecretSpec checks if the provided spec is valid.
func getDynamicSecretSpec(jsonSpec *apiextensions.JSON) (*genv1alpha1.BeyondTrustSecretsDynamicSecret, error) {
	if jsonSpec == nil {
		return nil, errors.New(errNoSpec)
	}

	spec, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, fmt.Errorf(errParseSpec, err)
	}
	if spec == nil || spec.Spec.Provider == nil {
		return nil, errors.New(errMissingConfig)
	}
	if spec.Spec.Provider.FolderPath == "" {
		return nil, errors.New(errNoPath)
	}

	return spec, nil
}

// parseSpec unmarshals the JSON spec into a BeyondTrustSecretsDynamicSecret struct.
func parseSpec(data []byte) (*genv1alpha1.BeyondTrustSecretsDynamicSecret, error) {
	var spec genv1alpha1.BeyondTrustSecretsDynamicSecret
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, err
	}
	return &spec, nil
}

// Parse the path to extract folder and secret name.
// Path format: "folder/subfolder/secretname" or just "secretname".
func parsePath(fullPath string) (*string, string) {
	var folderPath *string
	var secretName string

	lastSlash := strings.LastIndex(fullPath, "/")
	if lastSlash >= 0 {
		folder := fullPath[:lastSlash]
		folderPath = &folder
		secretName = fullPath[lastSlash+1:]
	} else {
		secretName = fullPath
	}
	return folderPath, secretName
}

// Convert generatedSecret to map[string][]byte.
func convertToByteMap(generatedSecret *btsutil.GeneratedSecret) map[string][]byte {
	out := make(map[string][]byte)

	// Defensive nil check
	if generatedSecret == nil {
		return out
	}

	out["accessKeyId"] = []byte(generatedSecret.AccessKeyID)
	out["secretAccessKey"] = []byte(generatedSecret.SecretAccessKey)
	out["leaseId"] = []byte(generatedSecret.LeaseID)
	out["expiration"] = []byte(generatedSecret.Expiration)

	if generatedSecret.SessionToken != "" {
		out["sessionToken"] = []byte(generatedSecret.SessionToken)
	}

	return out
}

// NewGenerator creates a new BeyondTrustSecrets generator instance.
func NewGenerator() genv1alpha1.Generator {
	return &Generator{}
}

// Kind returns the generator kind string.
func Kind() string {
	return string(genv1alpha1.GeneratorKindBeyondTrustSecretsDynamicSecret)
}
