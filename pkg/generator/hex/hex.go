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

package hex

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

const (
	errNoSpec    = "no config spec provided"
	errParseSpec = "unable to parse spec: %w"
	errGenerate  = "unable to generate hex string: %w"
)

// Generator implements the GeneratorInterface for hex strings.
type Generator struct{}

// generateFunc is used for dependency injection in tests.
type generateFunc func(length int) ([]byte, error)

// Generate creates a new hex string based on the provided spec.
func (g *Generator) Generate(_ context.Context, jsonSpec *apiextensions.JSON, _ client.Client, _ string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	return g.generate(jsonSpec, generateSecureBytes)
}

// Cleanup implements the GeneratorInterface.
func (g *Generator) Cleanup(_ context.Context, _ *apiextensions.JSON, _ genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

func (g *Generator) generate(jsonSpec *apiextensions.JSON, genBytes generateFunc) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}

	spec, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}

	if spec.Spec.Length < 1 {
		return nil, nil, fmt.Errorf("length must be greater than 0")
	}

	// Calculate number of random bytes needed (length/2 rounded up).
	byteLen := (spec.Spec.Length + 1) / 2
	randomBytes, err := genBytes(byteLen)
	if err != nil {
		return nil, nil, fmt.Errorf(errGenerate, err)
	}

	// Convert to hex string.
	hexStr := hex.EncodeToString(randomBytes)

	// Trim to exact length if odd number was requested.
	if len(hexStr) > spec.Spec.Length {
		hexStr = hexStr[:spec.Spec.Length]
	}

	// Convert to uppercase if requested.
	if spec.Spec.Uppercase {
		hexStr = strings.ToUpper(hexStr)
	}

	// Add prefix if specified.
	if spec.Spec.Prefix != "" {
		hexStr = spec.Spec.Prefix + hexStr
	}

	return map[string][]byte{
		"hex": []byte(hexStr),
	}, nil, nil
}

// generateSecureBytes generates cryptographically secure random bytes.
func generateSecureBytes(length int) ([]byte, error) {
	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return randomBytes, nil
}

func parseSpec(data []byte) (*genv1alpha1.Hex, error) {
	var spec genv1alpha1.Hex
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func init() {
	genv1alpha1.Register(genv1alpha1.HexKind, &Generator{})
}