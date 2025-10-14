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

// Package gcr provides functionality for generating authentication tokens for Google Container Registry.
package gcr

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"golang.org/x/oauth2"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/esutils/resolvers"
	"github.com/external-secrets/external-secrets/pkg/provider/gcp/secretmanager"
)

// Generator implements GCR token generation functionality.
type Generator struct{}

const (
	defaultLoginUsername = `oauth2accesstoken`

	errNoSpec    = "no config spec provided"
	errParseSpec = "unable to parse spec: %w"
	errGetToken  = "unable to get authorization token: %w"
)

// Generate creates an authentication token for Google Container Registry.
// It retrieves the token using the GCP credentials and returns it in the expected format.
func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	return g.generate(
		ctx,
		jsonSpec,
		kube,
		namespace,
		secretmanager.NewTokenSource,
	)
}

// Cleanup performs any necessary cleanup after token generation.
func (g *Generator) Cleanup(_ context.Context, _ *apiextensions.JSON, _ genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

func (g *Generator) generate(
	ctx context.Context,
	jsonSpec *apiextensions.JSON,
	kube client.Client,
	namespace string,
	tokenSource tokenSourceFunc) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}
	ts, err := tokenSource(ctx, esv1.GCPSMAuth{
		SecretRef:                  (*esv1.GCPSMAuthSecretRef)(res.Spec.Auth.SecretRef),
		WorkloadIdentity:           (*esv1.GCPWorkloadIdentity)(res.Spec.Auth.WorkloadIdentity),
		WorkloadIdentityFederation: res.Spec.Auth.WorkloadIdentityFederation,
	}, res.Spec.ProjectID, resolvers.EmptyStoreKind, kube, namespace)
	if err != nil {
		return nil, nil, err
	}
	token, err := ts.Token()
	if err != nil {
		return nil, nil, err
	}
	exp := strconv.FormatInt(token.Expiry.UTC().Unix(), 10)
	return map[string][]byte{
		"username": []byte(defaultLoginUsername),
		"password": []byte(token.AccessToken),
		"expiry":   []byte(exp),
	}, nil, nil
}

type tokenSourceFunc func(ctx context.Context, auth esv1.GCPSMAuth, projectID string, storeKind string, kube client.Client, namespace string) (oauth2.TokenSource, error)

func parseSpec(data []byte) (*genv1alpha1.GCRAccessToken, error) {
	var spec genv1alpha1.GCRAccessToken
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func init() {
	genv1alpha1.Register(genv1alpha1.GCRAccessTokenKind, &Generator{})
}
