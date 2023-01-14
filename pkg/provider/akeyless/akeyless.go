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
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/akeylesslabs/akeyless-go/v2"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	defaultAPIUrl = "https://api.akeyless.io"
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
	corev1    typedcorev1.CoreV1Interface
	namespace string

	akeylessGwAPIURL string
	RestAPI          *akeyless.V2ApiService
}

type Akeyless struct {
	Client akeylessVaultInterface
	url    string
}

type akeylessVaultInterface interface {
	GetSecretByType(secretName, token string, version int32) (string, error)
	TokenFromSecretRef(ctx context.Context) (string, error)
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

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) error {
	storeSpec := store.GetSpec()
	akeylessSpec := storeSpec.Provider.Akeyless

	akeylessGWApiURL := akeylessSpec.AkeylessGWApiURL

	if akeylessGWApiURL != nil && *akeylessGWApiURL != "" {
		url, err := url.Parse(*akeylessGWApiURL)
		if err != nil {
			return fmt.Errorf(errInvalidAkeylessURL)
		}

		if url.Host == "" {
			return fmt.Errorf(errInvalidAkeylessURL)
		}
	}
	if akeylessSpec.Auth.KubernetesAuth != nil {
		if akeylessSpec.Auth.KubernetesAuth.ServiceAccountRef != nil {
			if err := utils.ValidateReferentServiceAccountSelector(store, *akeylessSpec.Auth.KubernetesAuth.ServiceAccountRef); err != nil {
				return fmt.Errorf(errInvalidKubeSA, err)
			}
		}
		if akeylessSpec.Auth.KubernetesAuth.SecretRef != nil {
			err := utils.ValidateSecretSelector(store, *akeylessSpec.Auth.KubernetesAuth.SecretRef)
			if err != nil {
				return err
			}
		}

		if akeylessSpec.Auth.KubernetesAuth.AccessID == "" {
			return fmt.Errorf("missing kubernetes auth-method access-id")
		}

		if akeylessSpec.Auth.KubernetesAuth.K8sConfName == "" {
			return fmt.Errorf("missing kubernetes config name")
		}
		return nil
	}

	accessID := akeylessSpec.Auth.SecretRef.AccessID
	err := utils.ValidateSecretSelector(store, accessID)
	if err != nil {
		return err
	}

	if accessID.Name == "" {
		return fmt.Errorf(errInvalidAkeylessAccessIDName)
	}

	if accessID.Key == "" {
		return fmt.Errorf(errInvalidAkeylessAccessIDKey)
	}

	accessType := akeylessSpec.Auth.SecretRef.AccessType
	err = utils.ValidateSecretSelector(store, accessType)
	if err != nil {
		return err
	}

	accessTypeParam := akeylessSpec.Auth.SecretRef.AccessTypeParam
	err = utils.ValidateSecretSelector(store, accessTypeParam)
	if err != nil {
		return err
	}

	return nil
}

func newClient(_ context.Context, store esv1beta1.GenericStore, kube client.Client, corev1 typedcorev1.CoreV1Interface, namespace string) (esv1beta1.SecretsClient, error) {
	akl := &akeylessBase{
		kube:      kube,
		store:     store,
		namespace: namespace,
		corev1:    corev1,
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

	RestAPIClient := akeyless.NewAPIClient(&akeyless.Configuration{
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

func (a *Akeyless) Close(ctx context.Context) error {
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

func (a *Akeyless) PushSecret(ctx context.Context, value []byte, remoteRef esv1beta1.PushRemoteRef) error {
	return fmt.Errorf("not implemented")
}

func (a *Akeyless) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushRemoteRef) error {
	return fmt.Errorf("not implemented")
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
	value, err := a.Client.GetSecretByType(ref.Key, token, version)
	if err != nil {
		return nil, err
	}
	return []byte(value), nil
}

// Empty GetAllSecrets.
func (a *Akeyless) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return nil, fmt.Errorf("GetAllSecrets not implemented")
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
