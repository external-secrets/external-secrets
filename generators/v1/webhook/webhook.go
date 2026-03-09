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

// Package webhook provides functionality for generating secrets through webhook calls.
package webhook

import (
	"context"
	"encoding/json"
	"fmt"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	"github.com/external-secrets/external-secrets/providers/v1/webhook/pkg/webhook"
)

// Webhook represents a generator that calls external webhooks to generate secrets.
type Webhook struct {
	wh  webhook.Webhook
	url string
}

// Generate creates secrets by making webhook calls to external services.
func (w *Webhook) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kclient client.Client, ns string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	w.wh = webhook.Webhook{}
	w.wh.EnforceLabels = true
	w.wh.ClusterScoped = false
	provider, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse provider spec: %w", err)
	}
	w.wh.Namespace = ns
	w.url = provider.URL
	w.wh.Kube = kclient
	w.wh.HTTP, err = w.wh.GetHTTPClient(ctx, provider)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to prepare provider http client: %w", err)
	}
	data, err := w.wh.GetSecretMap(ctx, provider, nil)
	return data, nil, err
}

// Cleanup performs any necessary cleanup operations after secret generation.
func (w *Webhook) Cleanup(_ context.Context, _ *apiextensions.JSON, _ genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

func parseSpec(data []byte) (*webhook.Spec, error) {
	var spec genv1alpha1.Webhook
	err := json.Unmarshal(data, &spec)
	if err != nil {
		return nil, err
	}
	out := webhook.Spec{}
	d, err := json.Marshal(spec.Spec)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(d, &out)
	return &out, err
}

// NewGenerator creates a new Generator instance.
func NewGenerator() genv1alpha1.Generator {
	return &Webhook{}
}

// Kind returns the generator kind.
func Kind() string {
	return string(genv1alpha1.GeneratorKindWebhook)
}
