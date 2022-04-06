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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	tpl "text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/PaesslerAG/jsonpath"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/template/v2"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &WebHook{}
var _ esv1beta1.Provider = &Provider{}

// Provider satisfies the provider interface.
type Provider struct{}

type WebHook struct {
	kube      client.Client
	store     esv1beta1.GenericStore
	namespace string
	storeKind string
	http      *http.Client
	url       string
}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Webhook: &esv1beta1.WebhookProvider{},
	})
}

func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	whClient := &WebHook{
		kube:      kube,
		store:     store,
		namespace: namespace,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}
	provider, err := getProvider(store)
	if err != nil {
		return nil, err
	}
	whClient.url = provider.URL

	whClient.http, err = whClient.getHTTPClient(provider)
	if err != nil {
		return nil, err
	}
	return whClient, nil
}

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) error {
	return nil
}

func getProvider(store esv1beta1.GenericStore) (*esv1beta1.WebhookProvider, error) {
	spc := store.GetSpec()
	if spc == nil || spc.Provider == nil || spc.Provider.Webhook == nil {
		return nil, fmt.Errorf("missing store provider webhook")
	}
	return spc.Provider.Webhook, nil
}

func (w *WebHook) getStoreSecret(ctx context.Context, ref esmeta.SecretKeySelector) (*corev1.Secret, error) {
	ke := client.ObjectKey{
		Name:      ref.Name,
		Namespace: w.namespace,
	}
	if w.storeKind == esv1beta1.ClusterSecretStoreKind {
		if ref.Namespace == nil {
			return nil, fmt.Errorf("no namespace on ClusterSecretStore webhook secret %s", ref.Name)
		}
		ke.Namespace = *ref.Namespace
	}
	secret := &corev1.Secret{}
	if err := w.kube.Get(ctx, ke, secret); err != nil {
		return nil, fmt.Errorf("failed to get clustersecretstore webhook secret %s: %w", ref.Name, err)
	}
	return secret, nil
}

// Empty GetAllSecrets.
func (w *WebHook) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return nil, fmt.Errorf("GetAllSecrets not implemented")
}

func (w *WebHook) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	provider, err := getProvider(w.store)
	if err != nil {
		return nil, fmt.Errorf("failed to get store: %w", err)
	}
	result, err := w.getWebhookData(ctx, provider, ref)
	if err != nil {
		return nil, err
	}
	// Only parse as json if we have a jsonpath set
	if provider.Result.JSONPath != "" {
		jsondata := interface{}(nil)
		if err := yaml.Unmarshal(result, &jsondata); err != nil {
			return nil, fmt.Errorf("failed to parse response json: %w", err)
		}
		jsondata, err = jsonpath.Get(provider.Result.JSONPath, jsondata)
		if err != nil {
			return nil, fmt.Errorf("failed to get response path %s: %w", provider.Result.JSONPath, err)
		}
		jsonvalue, ok := jsondata.(string)
		if !ok {
			return nil, fmt.Errorf("failed to get response (wrong type: %T)", jsondata)
		}
		return []byte(jsonvalue), nil
	}

	return result, nil
}

func (w *WebHook) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	provider, err := getProvider(w.store)
	if err != nil {
		return nil, fmt.Errorf("failed to get store: %w", err)
	}
	result, err := w.getWebhookData(ctx, provider, ref)
	if err != nil {
		return nil, err
	}

	// We always want json here, so just parse it out
	jsondata := interface{}(nil)
	if err := yaml.Unmarshal(result, &jsondata); err != nil {
		return nil, fmt.Errorf("failed to parse response json: %w", err)
	}
	// Get subdata via jsonpath, if given
	if provider.Result.JSONPath != "" {
		jsondata, err = jsonpath.Get(provider.Result.JSONPath, jsondata)
		if err != nil {
			return nil, fmt.Errorf("failed to get response path %s: %w", provider.Result.JSONPath, err)
		}
	}
	// If the value is a string, try to parse it as json
	jsonstring, ok := jsondata.(string)
	if ok {
		// This could also happen if the response was a single json-encoded string
		// but that is an extremely unlikely scenario
		if err := yaml.Unmarshal([]byte(jsonstring), &jsondata); err != nil {
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

func (w *WebHook) getTemplateData(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef, secrets []esv1beta1.WebhookSecret) (map[string]map[string]string, error) {
	data := map[string]map[string]string{
		"remoteRef": {
			"key":      url.QueryEscape(ref.Key),
			"version":  url.QueryEscape(ref.Version),
			"property": url.QueryEscape(ref.Property),
		},
	}
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

func (w *WebHook) getWebhookData(ctx context.Context, provider *esv1beta1.WebhookProvider, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if w.http == nil {
		return nil, fmt.Errorf("http client not initialized")
	}
	data, err := w.getTemplateData(ctx, ref, provider.Secrets)
	if err != nil {
		return nil, err
	}
	method := provider.Method
	if method == "" {
		method = http.MethodGet
	}
	url, err := executeTemplateString(provider.URL, data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}
	body, err := executeTemplate(provider.Body, data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	for hKey, hValueTpl := range provider.Headers {
		hValue, err := executeTemplateString(hValueTpl, data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse header %s: %w", hKey, err)
		}
		req.Header.Add(hKey, hValue)
	}

	resp, err := w.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call endpoint: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("endpoint gave error %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func (w *WebHook) getHTTPClient(provider *esv1beta1.WebhookProvider) (*http.Client, error) {
	client := &http.Client{}
	if provider.Timeout != nil {
		client.Timeout = provider.Timeout.Duration
	}
	if len(provider.CABundle) == 0 && provider.CAProvider == nil {
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

func (w *WebHook) getCACertPool(provider *esv1beta1.WebhookProvider) (*x509.CertPool, error) {
	caCertPool := x509.NewCertPool()
	if len(provider.CABundle) > 0 {
		ok := caCertPool.AppendCertsFromPEM(provider.CABundle)
		if !ok {
			return nil, fmt.Errorf("failed to append cabundle")
		}
	}

	if provider.CAProvider != nil && w.storeKind == esv1beta1.ClusterSecretStoreKind && provider.CAProvider.Namespace == nil {
		return nil, fmt.Errorf("missing namespace on CAProvider secret")
	}

	if provider.CAProvider != nil {
		var cert []byte
		var err error

		switch provider.CAProvider.Type {
		case esv1beta1.WebhookCAProviderTypeSecret:
			cert, err = w.getCertFromSecret(provider)
		case esv1beta1.WebhookCAProviderTypeConfigMap:
			cert, err = w.getCertFromConfigMap(provider)
		default:
			err = fmt.Errorf("unknown caprovider type: %s", provider.CAProvider.Type)
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

func (w *WebHook) getCertFromSecret(provider *esv1beta1.WebhookProvider) ([]byte, error) {
	secretRef := esmeta.SecretKeySelector{
		Name: provider.CAProvider.Name,
		Key:  provider.CAProvider.Key,
	}

	if provider.CAProvider.Namespace != nil {
		secretRef.Namespace = provider.CAProvider.Namespace
	}

	ctx := context.Background()
	res, err := w.secretKeyRef(ctx, &secretRef)
	if err != nil {
		return nil, err
	}

	return []byte(res), nil
}

func (w *WebHook) secretKeyRef(ctx context.Context, secretRef *esmeta.SecretKeySelector) (string, error) {
	secret := &corev1.Secret{}
	ref := client.ObjectKey{
		Namespace: w.namespace,
		Name:      secretRef.Name,
	}
	if (w.storeKind == esv1beta1.ClusterSecretStoreKind) &&
		(secretRef.Namespace != nil) {
		ref.Namespace = *secretRef.Namespace
	}
	err := w.kube.Get(ctx, ref, secret)
	if err != nil {
		return "", err
	}

	keyBytes, ok := secret.Data[secretRef.Key]
	if !ok {
		return "", err
	}

	value := string(keyBytes)
	valueStr := strings.TrimSpace(value)
	return valueStr, nil
}

func (w *WebHook) getCertFromConfigMap(provider *esv1beta1.WebhookProvider) ([]byte, error) {
	objKey := client.ObjectKey{
		Name: provider.CAProvider.Name,
	}

	if provider.CAProvider.Namespace != nil {
		objKey.Namespace = *provider.CAProvider.Namespace
	}

	configMapRef := &corev1.ConfigMap{}
	ctx := context.Background()
	err := w.kube.Get(ctx, objKey, configMapRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get caprovider secret %s: %w", objKey.Name, err)
	}

	val, ok := configMapRef.Data[provider.CAProvider.Key]
	if !ok {
		return nil, fmt.Errorf("failed to get caprovider configmap %s -> %s", objKey.Name, provider.CAProvider.Key)
	}

	return []byte(val), nil
}

func (w *WebHook) Close(ctx context.Context) error {
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
	urlt, err := tpl.New("webhooktemplate").Funcs(sprig.TxtFuncMap()).Funcs(template.FuncMap()).Parse(tmpl)
	if err != nil {
		return result, err
	}
	if err := urlt.Execute(&result, data); err != nil {
		return result, err
	}
	return result, nil
}
