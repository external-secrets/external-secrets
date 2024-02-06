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
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	tpl "text/template"

	"github.com/PaesslerAG/jsonpath"
	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/template/v2"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

type Webhook struct {
	kube      client.Client
	namespace string
	storeKind string
	http      *http.Client
	url       string
}

func (w *Webhook) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kclient client.Client, ns string) (map[string][]byte, error) {
	provider, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, err
	}
	w.namespace = ns
	w.url = provider.Spec.URL
	w.kube = kclient
	w.http, err = w.getHTTPClient(provider)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get store: %w", err)
	}
	result, err := w.getWebhookData(ctx, provider)
	if err != nil {
		return nil, err
	}
	// We always want json here, so just parse it out
	jsondata := interface{}(nil)
	if err := json.Unmarshal(result, &jsondata); err != nil {
		return nil, fmt.Errorf("failed to parse response json: %w", err)
	}
	// Get subdata via jsonpath, if given
	if provider.Spec.Result.JSONPath != "" {
		jsondata, err = jsonpath.Get(provider.Spec.Result.JSONPath, jsondata)
		if err != nil {
			return nil, fmt.Errorf("failed to get response path %s: %w", provider.Spec.Result.JSONPath, err)
		}
	}
	// If the value is a string, try to parse it as json
	jsonstring, ok := jsondata.(string)
	if ok {
		// This could also happen if the response was a single json-encoded string
		// but that is an extremely unlikely scenario
		if err := json.Unmarshal([]byte(jsonstring), &jsondata); err != nil {
			return nil, fmt.Errorf("failed to parse response json from jsonpath: %w", err)
		}
	}
	// Use the data as a key-value map
	jsonvalue, ok := jsondata.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to get response (wrong type: %T)", jsondata)
	}

	// Change the map of generic objects to a map of byte arrays
	values := make(map[string][]byte)
	for rKey, rValue := range jsonvalue {
		jVal, ok := rValue.(string)
		if !ok {
			return nil, fmt.Errorf("failed to get response (wrong type in key '%s': %T)", rKey, rValue)
		}
		values[rKey] = []byte(jVal)
	}
	return values, nil
}

func parseSpec(data []byte) (*genv1alpha1.Webhook, error) {
	var spec genv1alpha1.Webhook
	err := json.Unmarshal(data, &spec)
	return &spec, err
}

func init() {
	genv1alpha1.Register(genv1alpha1.WebhookKind, &Webhook{})
}

func (w *Webhook) getStoreSecret(ctx context.Context, ref genv1alpha1.SecretKeySelector) (*corev1.Secret, error) {
	ke := client.ObjectKey{
		Name:      ref.Name,
		Namespace: w.namespace,
	}
	secret := &corev1.Secret{}
	if err := w.kube.Get(ctx, ke, secret); err != nil {
		return nil, fmt.Errorf("failed to get clustersecretstore webhook secret %s: %w", ref.Name, err)
	}
	expected, ok := secret.Labels["generators.external-secrets.io/type"]
	if !ok {
		return nil, fmt.Errorf("secret does not contain needed label to be used on webhook generator")
	}
	if expected != "webhook" {
		return nil, fmt.Errorf("secret type is not 'webhook'")
	}
	return secret, nil
}

func (w *Webhook) getTemplateData(ctx context.Context, secrets []genv1alpha1.WebhookSecret) (map[string]map[string]string, error) {
	data := map[string]map[string]string{}
	for _, secref := range secrets {
		if _, ok := data[secref.Name]; !ok {
			data[secref.Name] = make(map[string]string)
		}
		secret, err := w.getStoreSecret(ctx, secref.SecretRef)
		if err != nil {
			return nil, err
		}
		for sKey, sVal := range secret.Data {
			data[secref.Name][sKey] = string(sVal)
		}
	}
	return data, nil
}

func (w *Webhook) getWebhookData(ctx context.Context, provider *genv1alpha1.Webhook) ([]byte, error) {
	if w.http == nil {
		return nil, fmt.Errorf("http client not initialized")
	}
	data, err := w.getTemplateData(ctx, provider.Spec.Secrets)
	if err != nil {
		return nil, err
	}
	method := provider.Spec.Method
	if method == "" {
		method = http.MethodGet
	}
	url, err := executeTemplateString(provider.Spec.URL, data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}
	body, err := executeTemplate(provider.Spec.Body, data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	for hKey, hValueTpl := range provider.Spec.Headers {
		hValue, err := executeTemplateString(hValueTpl, data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse header %s: %w", hKey, err)
		}
		req.Header.Add(hKey, hValue)
	}

	resp, err := w.http.Do(req)
	metrics.ObserveAPICall(constants.ProviderWebhook, constants.CallWebhookHTTPReq, err)
	if err != nil {
		return nil, fmt.Errorf("failed to call endpoint: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("endpoint gave error %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func (w *Webhook) getHTTPClient(provider *genv1alpha1.Webhook) (*http.Client, error) {
	client := &http.Client{}
	if provider.Spec.Timeout != nil {
		client.Timeout = provider.Spec.Timeout.Duration
	}
	if len(provider.Spec.CABundle) == 0 && provider.Spec.CAProvider == nil {
		// No need to process ca stuff if it is not there
		return client, nil
	}
	caCertPool, err := w.getCACertPool(provider)
	if err != nil {
		return nil, err
	}

	tlsConf := &tls.Config{
		RootCAs:    caCertPool,
		MinVersion: tls.VersionTLS12,
	}
	client.Transport = &http.Transport{TLSClientConfig: tlsConf}
	return client, nil
}

func (w *Webhook) getCACertPool(provider *genv1alpha1.Webhook) (*x509.CertPool, error) {
	caCertPool := x509.NewCertPool()
	if len(provider.Spec.CABundle) > 0 {
		ok := caCertPool.AppendCertsFromPEM(provider.Spec.CABundle)
		if !ok {
			return nil, fmt.Errorf("failed to append cabundle")
		}
	}

	if provider.Spec.CAProvider != nil {
		var cert []byte
		var err error

		switch provider.Spec.CAProvider.Type {
		case genv1alpha1.WebhookCAProviderTypeSecret:
			cert, err = w.getCertFromSecret(provider)
		case genv1alpha1.WebhookCAProviderTypeConfigMap:
			cert, err = w.getCertFromConfigMap(provider)
		default:
			err = fmt.Errorf("unknown caprovider type: %s", provider.Spec.CAProvider.Type)
		}

		if err != nil {
			return nil, err
		}

		ok := caCertPool.AppendCertsFromPEM(cert)
		if !ok {
			return nil, fmt.Errorf("failed to append cabundle")
		}
	}
	return caCertPool, nil
}

func (w *Webhook) getCertFromSecret(provider *genv1alpha1.Webhook) ([]byte, error) {
	secretRef := esmeta.SecretKeySelector{
		Name:      provider.Spec.CAProvider.Name,
		Namespace: &w.namespace,
		Key:       provider.Spec.CAProvider.Key,
	}

	if provider.Spec.CAProvider.Namespace != nil {
		secretRef.Namespace = provider.Spec.CAProvider.Namespace
	}

	ctx := context.Background()
	cert, err := resolvers.SecretKeyRef(
		ctx,
		w.kube,
		w.storeKind,
		w.namespace,
		&secretRef,
	)
	if err != nil {
		return nil, err
	}

	return []byte(cert), nil
}

func (w *Webhook) getCertFromConfigMap(provider *genv1alpha1.Webhook) ([]byte, error) {
	objKey := client.ObjectKey{
		Name: provider.Spec.CAProvider.Name,
	}

	if provider.Spec.CAProvider.Namespace != nil {
		objKey.Namespace = *provider.Spec.CAProvider.Namespace
	}

	configMapRef := &corev1.ConfigMap{}
	ctx := context.Background()
	err := w.kube.Get(ctx, objKey, configMapRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get caprovider secret %s: %w", objKey.Name, err)
	}

	val, ok := configMapRef.Data[provider.Spec.CAProvider.Key]
	if !ok {
		return nil, fmt.Errorf("failed to get caprovider configmap %s -> %s", objKey.Name, provider.Spec.CAProvider.Key)
	}

	return []byte(val), nil
}

func executeTemplateString(tmpl string, data map[string]map[string]string) (string, error) {
	result, err := executeTemplate(tmpl, data)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

func executeTemplate(tmpl string, data map[string]map[string]string) (bytes.Buffer, error) {
	var result bytes.Buffer
	if tmpl == "" {
		return result, nil
	}
	urlt, err := tpl.New("webhooktemplate").Funcs(template.FuncMap()).Parse(tmpl)
	if err != nil {
		return result, err
	}
	if err := urlt.Execute(&result, data); err != nil {
		return result, err
	}
	return result, nil
}
