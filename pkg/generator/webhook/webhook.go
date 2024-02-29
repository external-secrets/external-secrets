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

package webhook

import (
	"context"
	"encoding/json"
	"fmt"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/common/webhook"
)

type Webhook struct {
	wh  webhook.Webhook
	url string
}

func (w *Webhook) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kclient client.Client, ns string) (map[string][]byte, error) {
	w.wh.EnforceLabels = true
	w.wh.ClusterScoped = false
	provider, err := parseSpec(jsonSpec.Raw)
	w.wh = webhook.Webhook{}
	if err != nil {
		return nil, fmt.Errorf("failed to parse provider spec: %w", err)
	}
	w.wh.Namespace = ns
	w.url = provider.URL
	w.wh.Kube = kclient
	w.wh.HTTP, err = w.wh.GetHTTPClient(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare provider http client: %w", err)
	}
	return w.wh.GetSecretMap(ctx, provider, nil)
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

func init() {
	genv1alpha1.Register(genv1alpha1.WebhookKind, &Webhook{})
}
