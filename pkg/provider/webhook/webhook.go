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
	"strconv"
	"time"

	"github.com/PaesslerAG/jsonpath"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/common/webhook"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	errNotImplemented = "not implemented"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &WebHook{}
var _ esv1beta1.Provider = &Provider{}

// Provider satisfies the provider interface.
type Provider struct{}

type WebHook struct {
	wh        webhook.Webhook
	store     esv1beta1.GenericStore
	storeKind string
	url       string
}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Webhook: &esv1beta1.WebhookProvider{},
	})
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

func (p *Provider) NewClient(_ context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	wh := webhook.Webhook{
		Kube:      kube,
		Namespace: namespace,
	}
	whClient := &WebHook{
		store:     store,
		wh:        wh,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}
	if whClient.storeKind == esv1beta1.ClusterSecretStoreKind {
		whClient.wh.ClusterScoped = true
	}
	provider, err := getProvider(store)
	if err != nil {
		return nil, err
	}
	whClient.url = provider.URL

	whClient.wh.HTTP, err = whClient.wh.GetHTTPClient(provider)
	if err != nil {
		return nil, err
	}
	return whClient, nil
}

func (p *Provider) ValidateStore(_ esv1beta1.GenericStore) (admission.Warnings, error) {
	return nil, nil
}

func getProvider(store esv1beta1.GenericStore) (*webhook.Spec, error) {
	spc := store.GetSpec()
	if spc == nil || spc.Provider == nil || spc.Provider.Webhook == nil {
		return nil, fmt.Errorf("missing store provider webhook")
	}
	out := webhook.Spec{}
	d, err := json.Marshal(spc.Provider.Webhook)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(d, &out)
	return &out, err
}

func (w *WebHook) DeleteSecret(_ context.Context, _ esv1beta1.PushSecretRemoteRef) error {
	return fmt.Errorf(errNotImplemented)
}

func (w *WebHook) SecretExists(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, fmt.Errorf(errNotImplemented)
}

// Not Implemented PushSecret.
func (w *WebHook) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1beta1.PushSecretData) error {
	return fmt.Errorf(errNotImplemented)
}

// Empty GetAllSecrets.
func (w *WebHook) GetAllSecrets(_ context.Context, _ esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return nil, fmt.Errorf(errNotImplemented)
}

func (w *WebHook) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	provider, err := getProvider(w.store)
	if err != nil {
		return nil, fmt.Errorf("failed to get store: %w", err)
	}
	result, err := w.wh.GetWebhookData(ctx, provider, &ref)
	if err != nil {
		return nil, err
	}
	// Only parse as json if we have a jsonpath set
	data, err := w.wh.GetTemplateData(ctx, &ref, provider.Secrets)
	if err != nil {
		return nil, err
	}
	resultJSONPath, err := webhook.ExecuteTemplateString(provider.Result.JSONPath, data)
	if err != nil {
		return nil, err
	}
	if resultJSONPath != "" {
		jsondata := any(nil)
		if err := json.Unmarshal(result, &jsondata); err != nil {
			return nil, fmt.Errorf("failed to parse response json: %w", err)
		}
		jsondata, err = jsonpath.Get(resultJSONPath, jsondata)
		if err != nil {
			return nil, fmt.Errorf("failed to get response path %s: %w", resultJSONPath, err)
		}
		return extractSecretData(jsondata)
	}

	return result, nil
}

// tries to extract data from an any
// it is supposed to return a single value.
func extractSecretData(jsondata any) ([]byte, error) {
	switch val := jsondata.(type) {
	case bool:
		return []byte(strconv.FormatBool(val)), nil
	case nil:
		return []byte{}, nil
	case int:
		return []byte(strconv.Itoa(val)), nil
	case float64:
		return []byte(strconv.FormatFloat(val, 'f', 0, 64)), nil
	case []byte:
		return val, nil
	case string:
		return []byte(val), nil

	// due to backwards compatibility we must keep this!
	// in case we see a []something we pick the first element and return it
	case []any:
		if len(val) == 0 {
			return nil, fmt.Errorf("filter worked but didn't get any result")
		}
		return extractSecretData(val[0])

	// in case we encounter a map we serialize it instead of erroring out
	// The user should use that data from within a template and figure
	// out how to deal with it.
	case map[string]any:
		return json.Marshal(val)
	default:
		return nil, fmt.Errorf("failed to get response (wrong type: %T)", jsondata)
	}
}

func (w *WebHook) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	provider, err := getProvider(w.store)
	if err != nil {
		return nil, fmt.Errorf("failed to get store: %w", err)
	}
	return w.wh.GetSecretMap(ctx, provider, &ref)
}

func (w *WebHook) Close(_ context.Context) error {
	return nil
}

func (w *WebHook) Validate() (esv1beta1.ValidationResult, error) {
	timeout := 15 * time.Second
	url := w.url

	if err := utils.NetworkValidate(url, timeout); err != nil {
		return esv1beta1.ValidationResultError, err
	}
	return esv1beta1.ValidationResultReady, nil
}
