//Copyright External Secrets Inc. All Rights Reserved

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
	w.wh.HTTP, err = w.wh.GetHTTPClient(ctx, provider)
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
