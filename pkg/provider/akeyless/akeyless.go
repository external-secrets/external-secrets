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
	"strconv"
	"time"

	"github.com/akeylesslabs/akeyless-go/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

// NewClient constructs a new secrets client based on the provided store.
func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	return newClient(ctx, store, kube, namespace)
}

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) error {
	return nil
}

func newClient(_ context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	akl := &akeylessBase{
		kube:      kube,
		store:     store,
		namespace: namespace,
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
