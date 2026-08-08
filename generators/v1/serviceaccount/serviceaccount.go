/*
Copyright © The ESO Authors

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

// Package serviceaccount issues short-lived Kubernetes ServiceAccount tokens
// through the TokenRequest API.
package serviceaccount

import (
	"context"
	"errors"
	"fmt"
	"time"

	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

const (
	// keyToken holds the issued bearer token.
	keyToken = "token"
	// keyExpirationTimestamp holds the RFC3339 instant at which the API server
	// stops accepting the token. It is reported by the issuer, not echoed from
	// the request, because the issuer may shorten what was asked for.
	keyExpirationTimestamp = "expirationTimestamp"

	// tokenSubResource is the ServiceAccount subresource backing TokenRequest.
	tokenSubResource = "token"
)

var (
	errNoSpec = errors.New("no spec provided")
	// A namespaced generator that could name another namespace would issue a
	// token the caller has no claim to, so the field is refused outright rather
	// than quietly dropped.
	errNamespaceUnsupported = errors.New("serviceAccountRef.namespace is not supported: a token is always issued in the namespace the generator is evaluated in")
)

// Generator issues ServiceAccount tokens.
type Generator struct{}

// Generate issues a token for the referenced ServiceAccount and returns it
// together with the expiry the API server assigned.
func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errNoSpec
	}

	spec, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse spec: %w", err)
	}

	ref := spec.Spec.ServiceAccountRef
	if ref.Namespace != nil {
		return nil, nil, errNamespaceUnsupported
	}

	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ref.Name,
			Namespace: namespace,
		},
	}
	tokenRequest := &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			Audiences:         ref.Audiences,
			ExpirationSeconds: spec.Spec.ExpirationSeconds,
		},
	}

	// The injected client is used rather than a clientset built from ambient
	// config, so that the call honors the manager's rest config and stays
	// substitutable in tests.
	if err := kube.SubResource(tokenSubResource).Create(ctx, serviceAccount, tokenRequest); err != nil {
		return nil, nil, fmt.Errorf("failed to issue token for service account %q: %w", ref.Name, err)
	}

	return map[string][]byte{
		keyToken:               []byte(tokenRequest.Status.Token),
		keyExpirationTimestamp: []byte(tokenRequest.Status.ExpirationTimestamp.UTC().Format(time.RFC3339)),
	}, nil, nil
}

// Cleanup is a no-op: a ServiceAccount token cannot be revoked individually, it
// only expires, so there is no state to keep and nothing to undo.
func (g *Generator) Cleanup(_ context.Context, _ *apiextensions.JSON, _ genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

func parseSpec(data []byte) (*genv1alpha1.ServiceAccountToken, error) {
	var spec genv1alpha1.ServiceAccountToken
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

// NewGenerator creates a new Generator instance.
func NewGenerator() genv1alpha1.Generator {
	return &Generator{}
}

// Kind returns the generator kind.
func Kind() string {
	return string(genv1alpha1.GeneratorKindServiceAccountToken)
}
