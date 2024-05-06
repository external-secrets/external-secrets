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

package akeyless

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/akeylesslabs/akeyless-go/v3"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

const (
	defaultAPIUrl     = "https://api.akeyless.io"
	errNotImplemented = "not implemented"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &Akeyless{}
var _ esv1beta1.Provider = &Provider{}

// Provider satisfies the provider interface.
type Provider struct{}

// akeylessBase satisfies the provider.SecretsClient interface.
type akeylessBase struct {
	kube      client.Client
	store     esv1beta1.GenericStore
	storeKind string
	corev1    typedcorev1.CoreV1Interface
	namespace string

	akeylessGwAPIURL string
	RestAPI          *akeyless.V2ApiService
}

type Akeyless struct {
	Client akeylessVaultInterface
	url    string
}

type Item struct {
	ItemName    string `json:"item_name"`
	ItemType    string `json:"item_type"`
	LastVersion int32  `json:"last_version"`
}

type akeylessVaultInterface interface {
	GetSecretByType(ctx context.Context, secretName, token string, version int32) (string, error)
	TokenFromSecretRef(ctx context.Context) (string, error)
	ListSecrets(ctx context.Context, path, tag, token string) ([]string, error)
}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Akeyless: &esv1beta1.AkeylessProvider{},
	})
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

// NewClient constructs a new secrets client based on the provided store.
func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	// controller-runtime/client does not support TokenRequest or other subresource APIs
	// so we need to construct our own client and use it to fetch tokens
	// (for Kubernetes service account token auth)
	restCfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}

	return newClient(ctx, store, kube, clientset.CoreV1(), namespace)
}

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	akeylessSpec := storeSpec.Provider.Akeyless

	akeylessGWApiURL := akeylessSpec.AkeylessGWApiURL

	if akeylessGWApiURL != nil && *akeylessGWApiURL != "" {
		url, err := url.Parse(*akeylessGWApiURL)
		if err != nil {
			return nil, fmt.Errorf(errInvalidAkeylessURL)
		}

		if url.Host == "" {
			return nil, fmt.Errorf(errInvalidAkeylessURL)
		}
	}
	if akeylessSpec.Auth.KubernetesAuth != nil {
		if akeylessSpec.Auth.KubernetesAuth.ServiceAccountRef != nil {
			if err := utils.ValidateReferentServiceAccountSelector(store, *akeylessSpec.Auth.KubernetesAuth.ServiceAccountRef); err != nil {
				return nil, fmt.Errorf(errInvalidKubeSA, err)
			}
		}
		if akeylessSpec.Auth.KubernetesAuth.SecretRef != nil {
			err := utils.ValidateSecretSelector(store, *akeylessSpec.Auth.KubernetesAuth.SecretRef)
			if err != nil {
				return nil, err
			}
		}

		if akeylessSpec.Auth.KubernetesAuth.AccessID == "" {
			return nil, fmt.Errorf("missing kubernetes auth-method access-id")
		}

		if akeylessSpec.Auth.KubernetesAuth.K8sConfName == "" {
			return nil, fmt.Errorf("missing kubernetes config name")
		}
		return nil, nil
	}

	accessID := akeylessSpec.Auth.SecretRef.AccessID
	err := utils.ValidateSecretSelector(store, accessID)
	if err != nil {
		return nil, err
	}

	if accessID.Name == "" {
		return nil, fmt.Errorf(errInvalidAkeylessAccessIDName)
	}

	if accessID.Key == "" {
		return nil, fmt.Errorf(errInvalidAkeylessAccessIDKey)
	}

	accessType := akeylessSpec.Auth.SecretRef.AccessType
	err = utils.ValidateSecretSelector(store, accessType)
	if err != nil {
		return nil, err
	}

	accessTypeParam := akeylessSpec.Auth.SecretRef.AccessTypeParam
	err = utils.ValidateSecretSelector(store, accessTypeParam)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func newClient(_ context.Context, store esv1beta1.GenericStore, kube client.Client, corev1 typedcorev1.CoreV1Interface, namespace string) (esv1beta1.SecretsClient, error) {
	akl := &akeylessBase{
		kube:      kube,
		store:     store,
		namespace: namespace,
		corev1:    corev1,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}

	spec, err := GetAKeylessProvider(store)
	if err != nil {
		return nil, err
	}
	akeylessGwAPIURL := defaultAPIUrl
	if spec != nil && spec.AkeylessGWApiURL != nil && *spec.AkeylessGWApiURL != "" {
		akeylessGwAPIURL = getV2Url(*spec.AkeylessGWApiURL)
	}

	if spec.Auth == nil {
		return nil, fmt.Errorf("missing Auth in store config")
	}

	client, err := akl.getAkeylessHTTPClient(spec)
	if err != nil {
		return nil, err
	}

	RestAPIClient := akeyless.NewAPIClient(&akeyless.Configuration{
		HTTPClient: client,
		Servers: []akeyless.ServerConfiguration{
			{
				URL: akeylessGwAPIURL,
			},
		},
	}).V2Api

	akl.akeylessGwAPIURL = akeylessGwAPIURL
	akl.RestAPI = RestAPIClient
	return &Akeyless{Client: akl, url: akeylessGwAPIURL}, nil
}

func (a *Akeyless) Close(_ context.Context) error {
	return nil
}

func (a *Akeyless) Validate() (esv1beta1.ValidationResult, error) {
	timeout := 15 * time.Second
	url := a.url

	if err := utils.NetworkValidate(url, timeout); err != nil {
		return esv1beta1.ValidationResultError, err
	}

	return esv1beta1.ValidationResultReady, nil
}

func (a *Akeyless) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1beta1.PushSecretData) error {
	return fmt.Errorf(errNotImplemented)
}

func (a *Akeyless) DeleteSecret(_ context.Context, _ esv1beta1.PushSecretRemoteRef) error {
	return fmt.Errorf(errNotImplemented)
}

func (a *Akeyless) SecretExists(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, fmt.Errorf(errNotImplemented)
}

// Implements store.Client.GetSecret Interface.
// Retrieves a secret with the secret name defined in ref.Name.
func (a *Akeyless) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if utils.IsNil(a.Client) {
		return nil, fmt.Errorf(errUninitalizedAkeylessProvider)
	}

	token, err := a.Client.TokenFromSecretRef(ctx)
	if err != nil {
		return nil, err
	}
	version := int32(0)
	if ref.Version != "" {
		i, err := strconv.ParseInt(ref.Version, 10, 32)
		if err == nil {
			version = int32(i)
		}
	}
	value, err := a.Client.GetSecretByType(ctx, ref.Key, token, version)
	if err != nil {
		return nil, err
	}

	if ref.Property == "" {
		if value != "" {
			return []byte(value), nil
		}
		return nil, fmt.Errorf("invalid value received, found no value string : %s", ref.Key)
	}
	// We need to search if a given key with a . exists before using gjson operations.
	idx := strings.Index(ref.Property, ".")
	if idx > -1 {
		refProperty := strings.ReplaceAll(ref.Property, ".", "\\.")
		val := gjson.Get(value, refProperty)
		if val.Exists() {
			return []byte(val.String()), nil
		}
	}
	val := gjson.Get(value, ref.Property)
	if !val.Exists() {
		return nil, fmt.Errorf("key %s does not exist in value %s", ref.Property, ref.Key)
	}
	return []byte(val.String()), nil
}

// Implements store.Client.GetAllSecrets Interface.
// Retrieves a all secrets with defined in ref.Name or tags.
func (a *Akeyless) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	if utils.IsNil(a.Client) {
		return nil, fmt.Errorf(errUninitalizedAkeylessProvider)
	}

	searchPath := ""
	if ref.Path != nil {
		searchPath = *ref.Path
		if !strings.HasPrefix(searchPath, "/") {
			searchPath = "/" + searchPath
		}
		if !strings.HasSuffix(searchPath, "/") {
			searchPath += "/"
		}
	}
	token, err := a.Client.TokenFromSecretRef(ctx)
	if err != nil {
		return nil, err
	}

	if ref.Name != nil {
		potentialSecrets, err := a.Client.ListSecrets(ctx, searchPath, "", token)
		if err != nil {
			return nil, err
		}
		if len(potentialSecrets) == 0 {
			return nil, nil
		}
		return a.findSecretsFromName(ctx, potentialSecrets, *ref.Name, token)
	}
	if len(ref.Tags) > 0 {
		var potentialSecretsName []string
		for _, v := range ref.Tags {
			potentialSecrets, err := a.Client.ListSecrets(ctx, searchPath, v, token)
			if err != nil {
				return nil, err
			}
			if len(potentialSecrets) > 0 {
				potentialSecretsName = append(potentialSecretsName, potentialSecrets...)
			}
		}
		if len(potentialSecretsName) == 0 {
			return nil, nil
		}
		return a.getSecrets(ctx, potentialSecretsName, token)
	}

	return nil, errors.New("unexpected find operator")
}

func (a *Akeyless) getSecrets(ctx context.Context, candidates []string, token string) (map[string][]byte, error) {
	secrets := make(map[string][]byte)
	for _, name := range candidates {
		secretValue, err := a.Client.GetSecretByType(ctx, name, token, 0)
		if err != nil {
			return nil, err
		}
		if secretValue != "" {
			secrets[name] = []byte(secretValue)
		}
	}
	return secrets, nil
}

func (a *Akeyless) findSecretsFromName(ctx context.Context, candidates []string, ref esv1beta1.FindName, token string) (map[string][]byte, error) {
	secrets := make(map[string][]byte)
	matcher, err := find.New(ref)
	if err != nil {
		return nil, err
	}
	for _, name := range candidates {
		ok := matcher.MatchName(name)
		if ok {
			secretValue, err := a.Client.GetSecretByType(ctx, name, token, 0)
			if err != nil {
				return nil, err
			}
			if secretValue != "" {
				secrets[name] = []byte(secretValue)
			}
		}
	}
	return secrets, nil
}

// Implements store.Client.GetSecretMap Interface.
// New version of GetSecretMap.
func (a *Akeyless) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	if utils.IsNil(a.Client) {
		return nil, fmt.Errorf(errUninitalizedAkeylessProvider)
	}

	val, err := a.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}
	// Maps the json data to a string:string map
	kv := make(map[string]string)
	err = json.Unmarshal(val, &kv)
	if err != nil {
		return nil, fmt.Errorf(errJSONSecretUnmarshal, err)
	}

	// Converts values in K:V pairs into bytes, while leaving keys as strings
	secretData := make(map[string][]byte)
	for k, v := range kv {
		secretData[k] = []byte(v)
	}
	return secretData, nil
}

func (a *akeylessBase) getAkeylessHTTPClient(provider *esv1beta1.AkeylessProvider) (*http.Client, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	if len(provider.CABundle) == 0 && provider.CAProvider == nil {
		return client, nil
	}
	caCertPool, err := a.getCACertPool(provider)
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

func (a *akeylessBase) getCACertPool(provider *esv1beta1.AkeylessProvider) (*x509.CertPool, error) {
	caCertPool := x509.NewCertPool()
	if len(provider.CABundle) > 0 {
		pem, err := base64decode(provider.CABundle)
		if err != nil {
			pem = provider.CABundle
		}
		ok := caCertPool.AppendCertsFromPEM(pem)
		if !ok {
			return nil, fmt.Errorf("failed to append caBundle")
		}
	}

	if provider.CAProvider != nil &&
		a.storeKind == esv1beta1.ClusterSecretStoreKind &&
		provider.CAProvider.Namespace == nil {
		return nil, fmt.Errorf("missing namespace on caProvider secret")
	}

	if provider.CAProvider != nil {
		var cert []byte
		var err error

		switch provider.CAProvider.Type {
		case esv1beta1.CAProviderTypeSecret:
			cert, err = a.getCertFromSecret(provider)
		case esv1beta1.CAProviderTypeConfigMap:
			cert, err = a.getCertFromConfigMap(provider)
		default:
			err = fmt.Errorf("unknown CAProvider type: %s", provider.CAProvider.Type)
		}

		if err != nil {
			return nil, err
		}
		pem, err := base64decode(cert)
		if err != nil {
			pem = cert
		}
		ok := caCertPool.AppendCertsFromPEM(pem)
		if !ok {
			return nil, fmt.Errorf("failed to append caBundle")
		}
	}
	return caCertPool, nil
}

func (a *akeylessBase) getCertFromSecret(provider *esv1beta1.AkeylessProvider) ([]byte, error) {
	secretRef := esmeta.SecretKeySelector{
		Name: provider.CAProvider.Name,
		Key:  provider.CAProvider.Key,
	}

	if provider.CAProvider.Namespace != nil {
		secretRef.Namespace = provider.CAProvider.Namespace
	}

	ctx := context.Background()
	cert, err := resolvers.SecretKeyRef(ctx, a.kube, a.storeKind, a.namespace, &secretRef)
	if err != nil {
		return nil, err
	}

	return []byte(cert), nil
}

func (a *akeylessBase) getCertFromConfigMap(provider *esv1beta1.AkeylessProvider) ([]byte, error) {
	objKey := client.ObjectKey{
		Name: provider.CAProvider.Name,
	}

	if provider.CAProvider.Namespace != nil {
		objKey.Namespace = *provider.CAProvider.Namespace
	}

	configMapRef := &corev1.ConfigMap{}
	ctx := context.Background()
	err := a.kube.Get(ctx, objKey, configMapRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get caProvider secret %s: %w", objKey.Name, err)
	}

	val, ok := configMapRef.Data[provider.CAProvider.Key]
	if !ok {
		return nil, fmt.Errorf("failed to get caProvider configMap %s -> %s", objKey.Name, provider.CAProvider.Key)
	}

	return []byte(val), nil
}
