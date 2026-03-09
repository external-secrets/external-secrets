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

// Package uuid provides functionality for generating random UUIDs.
package uuid

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

// Generator implements random UUID generation functionality.
type Generator struct{}

type generateFunc func() (string, error)

// Generate creates a random UUID.
func (g *Generator) Generate(_ context.Context, jsonSpec *apiextensions.JSON, _ client.Client, _ string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	return g.generate(
		jsonSpec,
		generateUUID,
	)
}

// Cleanup performs any necessary cleanup after token generation.
func (g *Generator) Cleanup(_ context.Context, _ *apiextensions.JSON, _ genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

func (g *Generator) generate(_ *apiextensions.JSON, uuidGen generateFunc) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	uuid, err := uuidGen()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to generate UUID: %w", err)
	}
	return map[string][]byte{
		"uuid": []byte(uuid),
	}, nil, nil
}

func generateUUID() (string, error) {
	uuid := uuid.New()
	return uuid.String(), nil
}

// NewGenerator creates a new Generator instance.
func NewGenerator() genv1alpha1.Generator {
	return &Generator{}
}

// Kind returns the generator kind.
func Kind() string {
	return string(genv1alpha1.GeneratorKindUUID)
}
