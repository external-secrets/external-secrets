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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	tpl "text/template"

	"github.com/Azure/go-ntlmssp"
	"github.com/PaesslerAG/jsonpath"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/template/v2"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

type Webhook struct {
	Kube          client.Client
	Namespace     string
	StoreKind     string
	HTTP          *http.Client
	EnforceLabels bool
	ClusterScoped bool
}

func (w *Webhook) getStoreSecret(ctx context.Context, ref esmeta.SecretKeySelector) (*corev1.Secret, error) {
	ke := client.ObjectKey{
		Name:      ref.Name,
		Namespace: w.Namespace,
	}
	if w.ClusterScoped {
		if ref.Namespace == nil {
			return nil, fmt.Errorf("no namespace on ClusterScoped webhook secret %s", ref.Name)
		}
		ke.Namespace = *ref.Namespace
	}
	secret := &corev1.Secret{}
	if err := w.Kube.Get(ctx, ke, secret); err != nil {
		return nil, fmt.Errorf("failed to get clustersecretstore webhook secret %s: %w", ref.Name, err)
	}
	if w.EnforceLabels {
		expected, ok := secret.Labels["external-secrets.io/type"]
		if !ok {
			return nil, errors.New("secret does not contain needed label 'external-secrets.io/type: webhook'. Update secret label to use it with webhook")
		}
		if expected != "webhook" {
			return nil, errors.New("secret type is not 'webhook'")
		}
	}
	return secret, nil
}
func (w *Webhook) GetSecretMap(ctx context.Context, provider *Spec, ref *esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	result, err := w.GetWebhookData(ctx, provider, ref)
	if err != nil {
		return nil, err
	}
	// We always want json here, so just parse it out
	jsondata := any(nil)
	if err := json.Unmarshal(result, &jsondata); err != nil {
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
		if err := json.Unmarshal([]byte(jsonstring), &jsondata); err != nil {
			return nil, fmt.Errorf("failed to parse response json from jsonpath: %w", err)
		}
	}
	// Use the data as a key-value map
	jsonvalue, ok := jsondata.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("failed to get response (wrong type: %T)", jsondata)
	}
	// Change the map of generic objects to a map of byte arrays
	values := make(map[string][]byte)
	for rKey := range jsonvalue {
		values[rKey], err = utils.GetByteValueFromMap(jsonvalue, rKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get response for key '%s': %w", rKey, err)
		}
	}
	return values, nil
}

func (w *Webhook) GetTemplateData(ctx context.Context, ref *esv1beta1.ExternalSecretDataRemoteRef, secrets []Secret, urlEncode bool) (map[string]map[string]string, error) {
	data := map[string]map[string]string{}
	if ref != nil {
		if urlEncode {
			data["remoteRef"] = map[string]string{
				"key":      url.QueryEscape(ref.Key),
				"version":  url.QueryEscape(ref.Version),
				"property": url.QueryEscape(ref.Property),
			}
		} else {
			data["remoteRef"] = map[string]string{
				"key":      ref.Key,
				"version":  ref.Version,
				"property": ref.Property,
			}
		}
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

func (w *Webhook) GetWebhookData(ctx context.Context, provider *Spec, ref *esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if w.HTTP == nil {
		return nil, errors.New("http client not initialized")
	}

	// Parse store secrets
	escapedData, err := w.GetTemplateData(ctx, ref, provider.Secrets, true)
	if err != nil {
		return nil, err
	}
	rawData, err := w.GetTemplateData(ctx, ref, provider.Secrets, false)
	if err != nil {
		return nil, err
	}

	// set method
	method := provider.Method
	if method == "" {
		method = http.MethodGet
	}

	// set url
	url, err := ExecuteTemplateString(provider.URL, escapedData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}

	// set body
	body, err := ExecuteTemplate(provider.Body, rawData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse body: %w", err)
	}

	// form request
	req, err := http.NewRequestWithContext(ctx, method, url, &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// add extra headers
	for hKey, hValueTpl := range provider.Headers {
		hValue, err := ExecuteTemplateString(hValueTpl, rawData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse header %s: %w", hKey, err)
		}
		req.Header.Add(hKey, hValue)
	}

	// add explicit credentials for specified auth protocols
	// any Auth headers set here will overwrite manually set Auth in provider.Headers
	if provider.Auth != nil {
		//nolint:gocritic // singleCaseSwitch: we prefer to keep it as a switch for clarity
		switch {
		case provider.Auth.NTLM != nil:

			userSecretRef := provider.Auth.NTLM.UserName
			userSecret, err := w.getStoreSecret(ctx, userSecretRef)
			if err != nil {
				return nil, err
			}
			username := string(userSecret.Data[userSecretRef.Key])

			PasswordSecretRef := provider.Auth.NTLM.Password
			PasswordSecret, err := w.getStoreSecret(ctx, PasswordSecretRef)
			if err != nil {
				return nil, err
			}
			password := string(PasswordSecret.Data[PasswordSecretRef.Key])

			// This overwrites auth headers set by providers.headers
			req.SetBasicAuth(username, password)
		}
	}

	resp, err := w.HTTP.Do(req)
	metrics.ObserveAPICall(constants.ProviderWebhook, constants.CallWebhookHTTPReq, err)
	if err != nil {
		return nil, fmt.Errorf("failed to call endpoint: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil, esv1beta1.NoSecretError{}
	}

	if resp.StatusCode == http.StatusNotModified {
		return nil, esv1beta1.NotModifiedError{}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("endpoint gave error %s", resp.Status)
	}

	// return response body
	return io.ReadAll(resp.Body)
}

func (w *Webhook) GetHTTPClient(ctx context.Context, provider *Spec) (*http.Client, error) {
	client := &http.Client{}

	// add timeout to client if it is there
	if provider.Timeout != nil {
		client.Timeout = provider.Timeout.Duration
	}

	// add CA to client if it is there
	if len(provider.CABundle) > 0 || provider.CAProvider != nil {
		caCertPool, err := w.GetCACertPool(ctx, provider)
		if err != nil {
			return nil, err
		}

		tlsConf := &tls.Config{
			RootCAs:       caCertPool,
			MinVersion:    tls.VersionTLS12,
			Renegotiation: tls.RenegotiateOnceAsClient,
		}

		client.Transport = &http.Transport{TLSClientConfig: tlsConf}
	}
	// add authentication method if it s there
	if provider.Auth != nil {
		//nolint:gocritic // singleCaseSwitch: we prefer to keep it as a switch for clarity
		switch {
		case provider.Auth.NTLM != nil:

			fmt.Println("Webhook Provider: Using ntlm authentication")
			client.Transport =
				&ntlmssp.Negotiator{
					RoundTripper: &http.Transport{
						TLSNextProto: map[string]func(authority string, c *tls.Conn) http.RoundTripper{}, // Needed to disable HTTP/2

					},
				}

			// add additional auth methods here
		}
	}

	// return client with all add-ons
	return client, nil
}

func (w *Webhook) GetCACertPool(ctx context.Context, provider *Spec) (*x509.CertPool, error) {
	caCertPool := x509.NewCertPool()
	ca, err := utils.FetchCACertFromSource(ctx, utils.CreateCertOpts{
		CABundle:   provider.CABundle,
		CAProvider: provider.CAProvider,
		StoreKind:  w.StoreKind,
		Namespace:  w.Namespace,
		Client:     w.Kube,
	})
	if err != nil {
		return nil, err
	}
	ok := caCertPool.AppendCertsFromPEM(ca)
	if !ok {
		return nil, errors.New("failed to append cabundle")
	}

	return caCertPool, nil
}

func (w *Webhook) GetCertFromSecret(provider *Spec) ([]byte, error) {
	secretRef := esmeta.SecretKeySelector{
		Name:      provider.CAProvider.Name,
		Namespace: &w.Namespace,
		Key:       provider.CAProvider.Key,
	}

	if provider.CAProvider.Namespace != nil {
		secretRef.Namespace = provider.CAProvider.Namespace
	}

	ctx := context.Background()
	cert, err := resolvers.SecretKeyRef(
		ctx,
		w.Kube,
		w.StoreKind,
		w.Namespace,
		&secretRef,
	)
	if err != nil {
		return nil, err
	}

	return []byte(cert), nil
}

func (w *Webhook) GetCertFromConfigMap(provider *Spec) ([]byte, error) {
	objKey := client.ObjectKey{
		Name: provider.CAProvider.Name,
	}

	if provider.CAProvider.Namespace != nil {
		objKey.Namespace = *provider.CAProvider.Namespace
	}

	configMapRef := &corev1.ConfigMap{}
	ctx := context.Background()
	err := w.Kube.Get(ctx, objKey, configMapRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get caprovider secret %s: %w", objKey.Name, err)
	}

	val, ok := configMapRef.Data[provider.CAProvider.Key]
	if !ok {
		return nil, fmt.Errorf("failed to get caprovider configmap %s -> %s", objKey.Name, provider.CAProvider.Key)
	}

	return []byte(val), nil
}

func ExecuteTemplateString(tmpl string, data map[string]map[string]string) (string, error) {
	result, err := ExecuteTemplate(tmpl, data)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

func ExecuteTemplate(tmpl string, data map[string]map[string]string) (bytes.Buffer, error) {
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
